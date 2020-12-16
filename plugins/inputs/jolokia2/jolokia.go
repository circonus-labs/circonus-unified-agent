package jolokia2

import (
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

func init() {
	inputs.Add("jolokia2_agent", func() cua.Input {
		return &JolokiaAgent{
			Metrics:               []MetricConfig{},
			DefaultFieldSeparator: ".",
		}
	})
	inputs.Add("jolokia2_proxy", func() cua.Input {
		return &JolokiaProxy{
			Metrics:               []MetricConfig{},
			DefaultFieldSeparator: ".",
		}
	})
}
