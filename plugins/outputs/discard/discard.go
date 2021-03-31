package discard

import (
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/outputs"
)

type Discard struct{}

func (d *Discard) Connect() error       { return nil }
func (d *Discard) Close() error         { return nil }
func (d *Discard) SampleConfig() string { return "" }
func (d *Discard) Description() string  { return "Send metrics to nowhere at all" }
func (d *Discard) Write(metrics []cua.Metric) (int, error) {
	return 0, nil
}

func init() {
	outputs.Add("discard", func() cua.Output { return &Discard{} })
}
