package shim

import "github.com/circonus-labs/circonus-unified-agent/cua"

// inputShim implements the MetricMaker interface.
type inputShim struct {
	Input cua.Input
}

func (i inputShim) LogName() string {
	return ""
}

func (i inputShim) MakeMetric(m cua.Metric) cua.Metric {
	return m // don't need to do anything to it.
}

func (i inputShim) Log() cua.Logger {
	return nil
}
