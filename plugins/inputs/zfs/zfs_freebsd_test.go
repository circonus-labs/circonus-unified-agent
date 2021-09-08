//go:build freebsd
// +build freebsd

package zfs

import (
	"context"
	"fmt"
	"testing"

	"github.com/circonus-labs/circonus-unified-agent/testutil"
	"github.com/stretchr/testify/require"
)

// $ zpool list -Hp -o name,health,size,alloc,free,fragmentation,capacity,dedupratio
var zpoolOutput = []string{
	"freenas-boot	ONLINE	30601641984	2022177280	28579464704	-	6	1.00x",
	"red1	ONLINE	8933531975680	1126164848640	7807367127040	8%	12	1.83x",
	"temp1	ONLINE	2989297238016	1626309320704	1362987917312	38%	54	1.28x",
	"temp2	ONLINE	2989297238016	626958278656	2362338959360	12%	20	1.00x",
}

func mockZpool() ([]string, error) {
	return zpoolOutput, nil
}

// $ zpool list -Hp -o name,health,size,alloc,free,fragmentation,capacity,dedupratio
var zpoolOutputUnavail = []string{
	"temp2	UNAVAIL	-	-	-	-	-	-",
}

func mockZpoolUnavail() ([]string, error) {
	return zpoolOutputUnavail, nil
}

// $ zfs list -Hp -o name,avail,used,usedsnap,usedds
var zdatasetOutput = []string{
	"zata    10741741326336  8564135526400   0       90112",
	"zata/home       10741741326336  2498560 212992  2285568",
	"zata/import     10741741326336  196608  81920   114688",
	"zata/storage    10741741326336  8556084379648   3601138999296   4954945380352",
}

func mockZdataset(properties []string) ([]string, error) {
	return zdatasetOutput, nil
}

// sysctl -q kstat.zfs.misc.arcstats

// sysctl -q kstat.zfs.misc.vdev_cache_stats
var kstatVdevCacheStatsOutput = []string{
	"kstat.zfs.misc.vdev_cache_stats.misses: 87789",
	"kstat.zfs.misc.vdev_cache_stats.hits: 465583",
	"kstat.zfs.misc.vdev_cache_stats.delegations: 6952",
}

// sysctl -q kstat.zfs.misc.zfetchstats
var kstatZfetchstatsOutput = []string{
	"kstat.zfs.misc.zfetchstats.max_streams: 0",
	"kstat.zfs.misc.zfetchstats.misses: 0",
	"kstat.zfs.misc.zfetchstats.hits: 0",
}

func mockSysctl(metric string) ([]string, error) {
	if metric == "vdev_cache_stats" {
		return kstatVdevCacheStatsOutput, nil
	}
	if metric == "zfetchstats" {
		return kstatZfetchstatsOutput, nil
	}
	return []string{}, fmt.Errorf("Invalid arg")
}

func TestZfsPoolMetrics(t *testing.T) {
	var acc testutil.Accumulator

	z := &Zfs{
		KstatMetrics: []string{"vdev_cache_stats"},
		sysctl:       mockSysctl,
		zpool:        mockZpool,
	}
	err := z.Gather(context.Background(), &acc)
	require.NoError(t, err)

	require.False(t, acc.HasMeasurement("zfs_pool"))
	acc.Metrics = nil

	z = &Zfs{
		KstatMetrics: []string{"vdev_cache_stats"},
		PoolMetrics:  true,
		sysctl:       mockSysctl,
		zpool:        mockZpool,
	}
	err = z.Gather(context.Background(), &acc)
	require.NoError(t, err)

	// one pool, all metrics
	tags := map[string]string{
		"pool":   "freenas-boot",
		"health": "ONLINE",
	}

	poolMetrics := getFreeNasBootPoolMetrics()

	acc.AssertContainsTaggedFields(t, "zfs_pool", poolMetrics, tags)
}

func TestZfsPoolMetrics_unavail(t *testing.T) {

	var acc testutil.Accumulator

	z := &Zfs{
		KstatMetrics: []string{"vdev_cache_stats"},
		sysctl:       mockSysctl,
		zpool:        mockZpoolUnavail,
	}
	err := z.Gather(context.Background(), &acc)
	require.NoError(t, err)

	require.False(t, acc.HasMeasurement("zfs_pool"))
	acc.Metrics = nil

	z = &Zfs{
		KstatMetrics: []string{"vdev_cache_stats"},
		PoolMetrics:  true,
		sysctl:       mockSysctl,
		zpool:        mockZpoolUnavail,
	}
	err = z.Gather(context.Background(), &acc)
	require.NoError(t, err)

	// one pool, UNAVAIL
	tags := map[string]string{
		"pool":   "temp2",
		"health": "UNAVAIL",
	}

	poolMetrics := getTemp2PoolMetrics()

	acc.AssertContainsTaggedFields(t, "zfs_pool", poolMetrics, tags)
}

func TestZfsDatasetMetrics(t *testing.T) {
	var acc testutil.Accumulator

	z := &Zfs{
		KstatMetrics: []string{"vdev_cache_stats"},
		sysctl:       mockSysctl,
		zdataset:     mockZdataset,
	}
	err := z.Gather(context.Background(), &acc)
	require.NoError(t, err)

	require.False(t, acc.HasMeasurement("zfs_dataset"))
	acc.Metrics = nil

	z = &Zfs{
		KstatMetrics:   []string{"vdev_cache_stats"},
		DatasetMetrics: true,
		sysctl:         mockSysctl,
		zdataset:       mockZdataset,
	}
	err = z.Gather(context.Background(), &acc)
	require.NoError(t, err)

	// one pool, all metrics
	tags := map[string]string{
		"dataset": "zata",
	}

	datasetMetrics := getZataDatasetMetrics()

	acc.AssertContainsTaggedFields(t, "zfs_dataset", datasetMetrics, tags)
}

func TestZfsGeneratesMetrics(t *testing.T) {
	var acc testutil.Accumulator

	z := &Zfs{
		KstatMetrics: []string{"vdev_cache_stats"},
		sysctl:       mockSysctl,
		zpool:        mockZpool,
	}
	err := z.Gather(context.Background(), &acc)
	require.NoError(t, err)

	// four pool, vdev_cache_stats metrics
	tags := map[string]string{
		"pools": "freenas-boot::red1::temp1::temp2",
	}
	intMetrics := getKstatMetricsVdevOnly()

	acc.AssertContainsTaggedFields(t, "zfs", intMetrics, tags)

	acc.Metrics = nil

	z = &Zfs{
		KstatMetrics: []string{"zfetchstats", "vdev_cache_stats"},
		sysctl:       mockSysctl,
		zpool:        mockZpool,
	}
	err = z.Gather(context.Background(), &acc)
	require.NoError(t, err)

	// four pool, vdev_cache_stats and zfetchstats metrics
	intMetrics = getKstatMetricsVdevAndZfetch()

	acc.AssertContainsTaggedFields(t, "zfs", intMetrics, tags)
}

func getFreeNasBootPoolMetrics() map[string]interface{} {
	return map[string]interface{}{
		"allocated":     int64(2022177280),
		"capacity":      int64(6),
		"dedupratio":    float64(1),
		"free":          int64(28579464704),
		"size":          int64(30601641984),
		"fragmentation": int64(0),
	}
}

func getTemp2PoolMetrics() map[string]interface{} {
	return map[string]interface{}{
		"size": int64(0),
	}
}

func getZataDatasetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"avail":    int64(10741741326336),
		"used":     int64(8564135526400),
		"usedsnap": int64(0),
		"usedds":   int64(90112),
	}
}

func getKstatMetricsVdevOnly() map[string]interface{} {
	return map[string]interface{}{
		"vdev_cache_stats_misses":      int64(87789),
		"vdev_cache_stats_hits":        int64(465583),
		"vdev_cache_stats_delegations": int64(6952),
	}
}

func getKstatMetricsVdevAndZfetch() map[string]interface{} {
	return map[string]interface{}{
		"vdev_cache_stats_misses":      int64(87789),
		"vdev_cache_stats_hits":        int64(465583),
		"vdev_cache_stats_delegations": int64(6952),
		"zfetchstats_max_streams":      int64(0),
		"zfetchstats_misses":           int64(0),
		"zfetchstats_hits":             int64(0),
	}
}
