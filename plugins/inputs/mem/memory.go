package mem

import (
	"context"
	"fmt"
	"runtime"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/system"
)

type Stats struct {
	ps       system.PS
	platform string
}

func (*Stats) Description() string {
	return "Read metrics about memory usage"
}

func (*Stats) SampleConfig() string {
	return `
  instance_id = "" # unique instance identifier (REQUIRED)
`
}

func (s *Stats) Init() error {
	s.platform = runtime.GOOS
	return nil
}

func (s *Stats) Gather(ctx context.Context, acc cua.Accumulator) error {
	vm, err := s.ps.VMStat()
	if err != nil {
		return fmt.Errorf("error getting virtual memory info: %w", err)
	}

	fields := map[string]interface{}{
		"total":             vm.Total,
		"available":         vm.Available,
		"used":              vm.Used,
		"used_percent":      100 * float64(vm.Used) / float64(vm.Total),
		"available_percent": 100 * float64(vm.Available) / float64(vm.Total),
	}

	switch s.platform {
	case "darwin":
		fields["active"] = vm.Active
		fields["free"] = vm.Free
		fields["inactive"] = vm.Inactive
		fields["wired"] = vm.Wired
	case "openbsd":
		fields["active"] = vm.Active
		fields["cached"] = vm.Cached
		fields["free"] = vm.Free
		fields["inactive"] = vm.Inactive
		fields["wired"] = vm.Wired
	case "freebsd":
		fields["active"] = vm.Active
		fields["buffered"] = vm.Buffers
		fields["cached"] = vm.Cached
		fields["free"] = vm.Free
		fields["inactive"] = vm.Inactive
		fields["laundry"] = vm.Laundry
		fields["wired"] = vm.Wired
	case "linux":
		fields["active"] = vm.Active
		fields["buffered"] = vm.Buffers
		fields["cached"] = vm.Cached
		fields["commit_limit"] = vm.CommitLimit
		fields["committed_as"] = vm.CommittedAS
		fields["dirty"] = vm.Dirty
		fields["free"] = vm.Free
		fields["high_free"] = vm.HighFree
		fields["high_total"] = vm.HighTotal
		fields["huge_pages_free"] = vm.HugePagesFree
		fields["huge_page_size"] = vm.HugePageSize
		fields["huge_pages_total"] = vm.HugePagesTotal
		fields["inactive"] = vm.Inactive
		fields["low_free"] = vm.LowFree
		fields["low_total"] = vm.LowTotal
		fields["mapped"] = vm.Mapped
		fields["page_tables"] = vm.PageTables
		fields["shared"] = vm.Shared
		fields["slab"] = vm.Slab
		fields["sreclaimable"] = vm.Sreclaimable
		fields["sunreclaim"] = vm.Sunreclaim
		fields["swap_cached"] = vm.SwapCached
		fields["swap_free"] = vm.SwapFree
		fields["swap_total"] = vm.SwapTotal
		fields["vmalloc_chunk"] = vm.VmallocChunk
		fields["vmalloc_total"] = vm.VmallocTotal
		fields["vmalloc_used"] = vm.VmallocUsed
		fields["write_back_tmp"] = vm.WriteBackTmp
		fields["write_back"] = vm.WriteBack
	}

	acc.AddGauge("mem", fields, nil)

	return nil
}

func init() {
	ps := system.NewSystemPS()
	inputs.Add("mem", func() cua.Input {
		return &Stats{ps: ps}
	})
}
