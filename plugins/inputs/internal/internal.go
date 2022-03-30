package internal

import (
	"context"
	"runtime"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/selfstat"
)

type Self struct {
	CollectMemstats  bool
	CollectSelfstats bool
}

func NewSelf() cua.Input {
	return &Self{
		CollectMemstats:  true,
		CollectSelfstats: true,
	}
}

var sampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)

  ## If true, collect cua memory stats.
  # collect_memstats = true

  ## if true, collect selfstats (per plugin)
  # collect_selfstats = true
`

func (s *Self) Description() string {
	return "Collect statistics about itself"
}

func (s *Self) SampleConfig() string {
	return sampleConfig
}

func (s *Self) Gather(ctx context.Context, acc cua.Accumulator) error {
	if s.CollectMemstats {
		m := &runtime.MemStats{}
		runtime.ReadMemStats(m)
		fields := map[string]interface{}{
			"alloc_bytes":       m.Alloc,      // bytes allocated and not yet freed
			"total_alloc_bytes": m.TotalAlloc, // bytes allocated (even if freed)
			"sys_bytes":         m.Sys,        // bytes obtained from system (sum of XxxSys below)
			"pointer_lookups":   m.Lookups,    // number of pointer lookups
			"mallocs":           m.Mallocs,    // number of mallocs
			"frees":             m.Frees,      // number of frees
			// Main allocation heap statistics.
			"heap_alloc_bytes":    m.HeapAlloc,    // bytes allocated and not yet freed (same as Alloc above)
			"heap_sys_bytes":      m.HeapSys,      // bytes obtained from system
			"heap_idle_bytes":     m.HeapIdle,     // bytes in idle spans
			"heap_in_use_bytes":   m.HeapInuse,    // bytes in non-idle span
			"heap_released_bytes": m.HeapReleased, // bytes released to the OS
			"heap_objects":        m.HeapObjects,  // total number of allocated objects
			"num_gc":              m.NumGC,
		}
		acc.AddFields("internal_memstats", fields, map[string]string{"__rollup": "false"})
	}

	if s.CollectSelfstats {
		goVersion := strings.TrimPrefix(runtime.Version(), "go")

		for _, m := range selfstat.Metrics() {
			if m.Name() == "internal_agent" {
				m.AddTag("go_version", goVersion)
			}
			m.AddTag("__rollup", "false")
			acc.AddFields(m.Name(), m.Fields(), m.Tags(), m.Time())
		}
	}

	return nil
}

func init() {
	inputs.Add("internal", NewSelf)
}
