package zfs

import (
	"github.com/circonus-labs/circonus-unified-agent/cua"
)

type Sysctl func(metric string) ([]string, error)
type Zpool func() ([]string, error)
type Zdataset func(properties []string) ([]string, error)

type Zfs struct {
	Log            cua.Logger `toml:"-"`
	zdataset       Zdataset   //nolint:structcheck,unused
	sysctl         Sysctl     //nolint:structcheck,unused
	zpool          Zpool      //nolint:structcheck,unused
	KstatPath      string
	ZpoolPath      string
	KstatMetrics   []string
	PoolMetrics    bool
	DatasetMetrics bool
}

var sampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)
  
  ## By default, gather zpool stats
  poolMetrics = true

  # ATTENTION LINUX USERS:
  # Because circonus-unified-agent normally runs as an unprivileged user, it may not be
  # able to run "zpool {status,list}" without root privileges, due to the
  # permissions on /dev/zfs.
  # This was addressed in ZFSonLinux 0.7.0 and later.
  # See https://github.com/zfsonlinux/zfs/issues/362 for a potential workaround
  # if your distribution does not support unprivileged access to /dev/zfs.

  ## Path for zpool command, the default is:
  # zpoolPath = "/usr/sbin/zpool"

  ## ZFS kstat path. Ignored on FreeBSD
  ## If not specified, then default is:
  # kstatPath = "/proc/spl/kstat/zfs"

  ## By default, agent gathers all zfs stats
  ## If not specified, then default is:
  # kstatMetrics = ["arcstats", "zfetchstats", "vdev_cache_stats"]
  ## For Linux, the default is:
  # kstatMetrics = ["abdstats", "arcstats", "dnodestats", "dbufcachestats",
  #   "dmu_tx", "fm", "vdev_mirror_stats", "zfetchstats", "zil"]
  ## By default, don't gather dataset metrics
  # datasetMetrics = false
`

func (z *Zfs) SampleConfig() string {
	return sampleConfig
}

func (z *Zfs) Description() string {
	return "Read metrics of ZFS from arcstats, zfetchstats, vdev_cache_stats, pools, and datasets"
}
