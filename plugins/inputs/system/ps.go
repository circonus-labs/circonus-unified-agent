package system

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

type PS interface {
	CPUTimes(perCPU, totalCPU bool) ([]cpu.TimesStat, error)
	DiskUsage(mountPointFilter []string, fstypeExclude []string) ([]*disk.UsageStat, []*disk.PartitionStat, error)
	NetIO() ([]net.IOCountersStat, error)
	NetProto() ([]net.ProtoCountersStat, error)
	DiskIO(names []string) (map[string]disk.IOCountersStat, error)
	VMStat() (*mem.VirtualMemoryStat, error)
	SwapStat() (*mem.SwapMemoryStat, error)
	NetConnections() ([]net.ConnectionStat, error)
	Temperature() ([]host.TemperatureStat, error)
}

type PSDiskDeps interface {
	Partitions(all bool) ([]disk.PartitionStat, error)
	OSGetenv(key string) string
	OSStat(name string) (os.FileInfo, error)
	PSDiskUsage(path string) (*disk.UsageStat, error)
}

// func add(acc cua.Accumulator,
// 	name string, val float64, tags map[string]string) {
// 	if val >= 0 {
// 		acc.AddFields(name, map[string]interface{}{"value": val}, tags)
// 	}
// }

func NewSystemPS() *SysPS {
	return &SysPS{&SysPSDisk{}}
}

type SysPS struct {
	PSDiskDeps
}

type SysPSDisk struct{}

func (s *SysPS) CPUTimes(perCPU, totalCPU bool) ([]cpu.TimesStat, error) {
	var cpuTimes []cpu.TimesStat
	if perCPU {
		if perCPUTimes, err := cpu.Times(true); err == nil {
			cpuTimes = append(cpuTimes, perCPUTimes...)
		} else {
			return nil, fmt.Errorf("cpu times (percpu): %w", err)
		}
	}
	if totalCPU {
		if totalCPUTimes, err := cpu.Times(false); err == nil {
			cpuTimes = append(cpuTimes, totalCPUTimes...)
		} else {
			return nil, fmt.Errorf("cpu times (tot): %w", err)
		}
	}
	return cpuTimes, nil
}

func (s *SysPS) DiskUsage(
	mountPointFilter []string,
	fstypeExclude []string,
) ([]*disk.UsageStat, []*disk.PartitionStat, error) {
	parts, err := s.Partitions(true)
	if err != nil {
		return nil, nil, err
	}

	// Make a "set" out of the filter slice
	mountPointFilterSet := make(map[string]bool)
	for _, filter := range mountPointFilter {
		mountPointFilterSet[filter] = true
	}
	fstypeExcludeSet := make(map[string]bool)
	for _, filter := range fstypeExclude {
		fstypeExcludeSet[filter] = true
	}
	paths := make(map[string]bool)
	for _, part := range parts {
		paths[part.Mountpoint] = true
	}

	// Autofs mounts indicate a potential mount, the partition will also be
	// listed with the actual filesystem when mounted.  Ignore the autofs
	// partition to avoid triggering a mount.
	fstypeExcludeSet["autofs"] = true

	usage := make([]*disk.UsageStat, 0, len(parts))
	partitions := make([]*disk.PartitionStat, 0, len(parts))
	hostMountPrefix := s.OSGetenv("HOST_MOUNT_PREFIX")

	for i := range parts {
		p := parts[i]

		if len(mountPointFilter) > 0 {
			// If the mount point is not a member of the filter set,
			// don't gather info on it.
			if _, ok := mountPointFilterSet[p.Mountpoint]; !ok {
				continue
			}
		}

		// If the mount point is a member of the exclude set,
		// don't gather info on it.
		if _, ok := fstypeExcludeSet[p.Fstype]; ok {
			continue
		}

		// If there's a host mount prefix, exclude any paths which conflict
		// with the prefix.
		if len(hostMountPrefix) > 0 &&
			!strings.HasPrefix(p.Mountpoint, hostMountPrefix) &&
			paths[hostMountPrefix+p.Mountpoint] {
			continue
		}

		du, err := s.PSDiskUsage(p.Mountpoint)
		if err != nil {
			continue
		}

		du.Path = filepath.Join("/", strings.TrimPrefix(p.Mountpoint, hostMountPrefix))
		du.Fstype = p.Fstype
		usage = append(usage, du)
		partitions = append(partitions, &p)
	}

	return usage, partitions, nil
}

func (s *SysPS) NetProto() ([]net.ProtoCountersStat, error) {
	return net.ProtoCounters(nil)
}

func (s *SysPS) NetIO() ([]net.IOCountersStat, error) {
	return net.IOCounters(true)
}

func (s *SysPS) NetConnections() ([]net.ConnectionStat, error) {
	return net.Connections("all")
}

func (s *SysPS) DiskIO(names []string) (map[string]disk.IOCountersStat, error) {
	m, err := disk.IOCounters(names...)
	if err != nil {
		if errors.Is(err, internal.ErrNotImplemented) {
			return nil, nil
		}
		return nil, fmt.Errorf("disk iocounters: %w", err)
	}

	return m, nil
}

func (s *SysPS) VMStat() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}

func (s *SysPS) SwapStat() (*mem.SwapMemoryStat, error) {
	return mem.SwapMemory()
}

func (s *SysPS) Temperature() ([]host.TemperatureStat, error) {
	temp, err := host.SensorsTemperatures()
	if err != nil {
		var hwerr *host.Warnings
		if errors.As(err, &hwerr) {
			return temp, fmt.Errorf("temp sensors: %w", err)
		}
	}
	return temp, nil
}

func (s *SysPSDisk) Partitions(all bool) ([]disk.PartitionStat, error) {
	return disk.Partitions(all)
}

func (s *SysPSDisk) OSGetenv(key string) string {
	return os.Getenv(key)
}

func (s *SysPSDisk) OSStat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (s *SysPSDisk) PSDiskUsage(path string) (*disk.UsageStat, error) {
	return disk.Usage(path)
}
