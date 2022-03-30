//go:build !windows
// +build !windows

package bcache

// bcache doesn't aim for Windows

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type Bcache struct {
	BcachePath string
	BcacheDevs []string
}

var sampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)

  ## Bcache sets path
  ## If not specified, then default is:
  bcachePath = "/sys/fs/bcache"

  ## By default, the agent gathers stats for all bcache devices
  ## Setting devices will restrict the stats to the specified
  ## bcache devices.
  bcacheDevs = ["bcache0"]
`

func (b *Bcache) SampleConfig() string {
	return sampleConfig
}

func (b *Bcache) Description() string {
	return "Read metrics of bcache from stats_total and dirty_data"
}

func getTags(bdev string) map[string]string {
	backingDevFile, _ := os.Readlink(bdev)
	backingDevPath := strings.Split(backingDevFile, "/")
	backingDev := backingDevPath[len(backingDevPath)-2]

	bcacheDevFile, _ := os.Readlink(bdev + "/dev")
	bcacheDevPath := strings.Split(bcacheDevFile, "/")
	bcacheDev := bcacheDevPath[len(bcacheDevPath)-1]

	return map[string]string{"backing_dev": backingDev, "bcache_dev": bcacheDev}
}

func prettyToBytes(v string) uint64 {
	var factors = map[string]uint64{
		"k": 1 << 10,
		"M": 1 << 20,
		"G": 1 << 30,
		"T": 1 << 40,
		"P": 1 << 50,
		"E": 1 << 60,
	}
	var factor uint64
	factor = 1
	prefix := v[len(v)-1:]
	if factors[prefix] != 0 {
		v = v[:len(v)-1]
		factor = factors[prefix]
	}
	result, _ := strconv.ParseFloat(v, 32)
	result *= float64(factor)

	return uint64(result)
}

func (b *Bcache) gatherBcache(bdev string, acc cua.Accumulator) error {
	tags := getTags(bdev)
	metrics, err := filepath.Glob(bdev + "/stats_total/*")
	if err != nil {
		return fmt.Errorf("glob: %w", err)
	}
	if len(metrics) == 0 {
		return errors.New("can't read any stats file")
	}
	file, err := os.ReadFile(bdev + "/dirty_data")
	if err != nil {
		return fmt.Errorf("readfile (%s): %w", bdev+"/dirty_data", err)
	}
	rawValue := strings.TrimSpace(string(file))
	value := prettyToBytes(rawValue)

	fields := make(map[string]interface{})
	fields["dirty_data"] = value

	for _, path := range metrics {
		key := filepath.Base(path)
		file, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("readfile (%s): %w", path, err)
		}
		rawValue := strings.TrimSpace(string(file))
		if key == "bypassed" {
			value := prettyToBytes(rawValue)
			fields[key] = value
		} else {
			value, _ := strconv.ParseUint(rawValue, 10, 64)
			fields[key] = value
		}
	}
	acc.AddFields("bcache", fields, tags)
	return nil
}

func (b *Bcache) Gather(ctx context.Context, acc cua.Accumulator) error {
	bcacheDevsChecked := make(map[string]bool)
	var restrictDevs bool
	if len(b.BcacheDevs) != 0 {
		restrictDevs = true
		for _, bcacheDev := range b.BcacheDevs {
			bcacheDevsChecked[bcacheDev] = true
		}
	}

	bcachePath := b.BcachePath
	if len(bcachePath) == 0 {
		bcachePath = "/sys/fs/bcache"
	}
	bdevs, _ := filepath.Glob(bcachePath + "/*/bdev*")
	if len(bdevs) < 1 {
		return errors.New("Can't find any bcache device")
	}
	for _, bdev := range bdevs {
		if restrictDevs {
			bcacheDev := getTags(bdev)["bcache_dev"]
			if !bcacheDevsChecked[bcacheDev] {
				continue
			}
		}
		_ = b.gatherBcache(bdev, acc)
	}
	return nil
}

func init() {
	inputs.Add("bcache", func() cua.Input {
		return &Bcache{}
	})
}
