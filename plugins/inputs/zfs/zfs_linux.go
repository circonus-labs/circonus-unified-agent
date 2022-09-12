//go:build linux
// +build linux

package zfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

const (
	unknown int = iota
	online
	degraded
	faulted
	offline
	removed
	unavail
)

type poolInfo struct {
	name       string
	ioFilename string
}

func getPools(kstatPath string) []poolInfo {
	pools := make([]poolInfo, 0)
	poolsDirs, _ := filepath.Glob(kstatPath + "/*/io")

	for _, poolDir := range poolsDirs {
		poolDirSplit := strings.Split(poolDir, "/")
		pool := poolDirSplit[len(poolDirSplit)-2]
		pools = append(pools, poolInfo{name: pool, ioFilename: poolDir})
	}

	return pools
}

func getTags(pools []poolInfo) map[string]string {
	var poolNames string

	for _, pool := range pools {
		if len(poolNames) != 0 {
			poolNames += "::"
		}
		poolNames += pool.name
	}

	return map[string]string{"pools": poolNames}
}

func gatherPoolStats(pool poolInfo, acc cua.Accumulator) error {
	lines, err := internal.ReadLines(pool.ioFilename)
	if err != nil {
		return fmt.Errorf("zfs pool stats (%s): %w", pool.ioFilename, err)
	}

	if len(lines) != 3 {
		return fmt.Errorf("zfs pool stats invalid #lines: %w", err)
	}

	keys := strings.Fields(lines[1])
	values := strings.Fields(lines[2])

	keyCount := len(keys)

	if keyCount != len(values) {
		return fmt.Errorf("Key and value count don't match Keys:%v Values:%v", keys, values)
	}

	tag := map[string]string{"pool": pool.name}
	fields := make(map[string]interface{})
	for i := 0; i < keyCount; i++ {
		value, err := strconv.ParseInt(values[i], 10, 64)
		if err != nil {
			return fmt.Errorf("zfs pool stats parseint (%s): %w", values[i], err)
		}
		fields[keys[i]] = value
	}
	acc.AddFields("zfs_pool", fields, tag)

	return nil
}

func (z *Zfs) Gather(ctx context.Context, acc cua.Accumulator) error {
	kstatMetrics := z.KstatMetrics
	if len(kstatMetrics) == 0 {
		// vdev_cache_stats is deprecated
		// xuio_stats are ignored because as of Sep-2016, no known
		// consumers of xuio exist on Linux
		kstatMetrics = []string{"abdstats", "arcstats", "dnodestats", "dbufcachestats",
			"dmu_tx", "fm", "vdev_mirror_stats", "zfetchstats", "zil"}
	}

	kstatPath := z.KstatPath
	if len(kstatPath) == 0 {
		kstatPath = "/proc/spl/kstat/zfs"
	}

	pools := getPools(kstatPath)
	tags := getTags(pools)

	if z.PoolMetrics {
		for _, pool := range pools {
			err := gatherPoolStats(pool, acc)
			if err != nil {
				return err
			}
		}
		_, err := z.gatherPoolListStats(acc)
		if err != nil {
			return err
		}
	}

	fields := make(map[string]interface{})
	for _, metric := range kstatMetrics {
		lines, err := internal.ReadLines(kstatPath + "/" + metric)
		if err != nil {
			continue
		}
		for i, line := range lines {
			if i == 0 || i == 1 {
				continue
			}
			if len(line) < 1 {
				continue
			}
			rawData := strings.Split(line, " ")
			key := metric + "_" + rawData[0]
			if metric == "zil" || metric == "dmu_tx" || metric == "dnodestats" {
				key = rawData[0]
			}
			rawValue := rawData[len(rawData)-1]
			value, _ := strconv.ParseInt(rawValue, 10, 64)
			fields[key] = value
		}
	}
	acc.AddFields("zfs", fields, tags)
	return nil
}

func init() {
	inputs.Add("zfs", func() cua.Input {
		return &Zfs{}
	})
}

func (z *Zfs) gatherPoolListStats(acc cua.Accumulator) (string, error) {
	if !z.PoolMetrics {
		return "", nil
	}

	zpoolpath := z.ZpoolPath
	if len(zpoolpath) == 0 {
		zpoolpath = "/usr/sbin/zpool"
	}

	lines, err := zpoolList(zpoolpath)
	if err != nil {
		return "", err
	}

	pools := []string{}
	for _, line := range lines {
		col := strings.Split(line, "\t")

		pools = append(pools, col[0])
	}

	for _, line := range lines {
		col := strings.Split(line, "\t")
		if len(col) != 8 {
			continue
		}

		tags := map[string]string{"pool": col[0]}
		fields := map[string]interface{}{}

		health := unknown
		switch col[1] {
		case "ONLINE":
			health = online
		case "DEGRADED":
			health = degraded
		case "FAULTED":
			health = faulted
		case " OFFLINE":
			health = offline
		case "REMOVED":
			health = removed
		case "UNAVAIL":
			health = unavail
		}
		fields["health"] = health

		if health == unavail {

			fields["size"] = int64(0)

		} else {

			size, err := strconv.ParseInt(col[2], 10, 64)
			if err != nil {
				return "", fmt.Errorf("Error parsing size: %w", err)
			}
			fields["size"] = size

			alloc, err := strconv.ParseInt(col[3], 10, 64)
			if err != nil {
				return "", fmt.Errorf("Error parsing allocation: %w", err)
			}
			fields["allocated"] = alloc

			free, err := strconv.ParseInt(col[4], 10, 64)
			if err != nil {
				return "", fmt.Errorf("Error parsing free: %w", err)
			}
			fields["free"] = free

			frag, err := strconv.ParseInt(strings.TrimSuffix(col[5], "%"), 10, 0)
			if err != nil { // This might be - for RO devs
				frag = 0
			}
			fields["fragmentation"] = frag

			capval, err := strconv.ParseInt(col[6], 10, 0)
			if err != nil {
				return "", fmt.Errorf("Error parsing capacity: %w", err)
			}
			fields["capacity"] = capval

			dedup, err := strconv.ParseFloat(strings.TrimSuffix(col[7], "x"), 32)
			if err != nil {
				return "", fmt.Errorf("Error parsing dedupratio: %w", err)
			}
			fields["dedupratio"] = dedup
		}

		acc.AddFields("zfs_pool", fields, tags)
	}

	return strings.Join(pools, "::"), nil
}

func run(command string, args ...string) ([]string, error) {
	cmd := exec.Command(command, args...)
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()

	stdout := strings.TrimSpace(outbuf.String())
	stderr := strings.TrimSpace(errbuf.String())

	var exitErr *exec.ExitError
	if err != nil {
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("%s error: %s: %w", command, stderr, exitErr)
		}
		return nil, fmt.Errorf("%s error: %w", command, err)
	}
	return strings.Split(stdout, "\n"), nil
}

func zpoolList(zpoolpath string) ([]string, error) {
	zpoolcmd := "zpool"
	if len(zpoolpath) != 0 {
		zpoolcmd = zpoolpath
	}
	return run(zpoolcmd, []string{"list", "-Hp", "-o", "name,health,size,alloc,free,fragmentation,capacity,dedupratio"}...)
}
