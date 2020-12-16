package pivot

import (
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/processors"
)

const (
	description  = "Rotate a single valued metric into a multi field metric"
	sampleConfig = `
  ## Tag to use for naming the new field.
  tag_key = "name"
  ## Field to use as the value of the new field.
  value_key = "value"
`
)

type Pivot struct {
	TagKey   string `toml:"tag_key"`
	ValueKey string `toml:"value_key"`
}

func (p *Pivot) SampleConfig() string {
	return sampleConfig
}

func (p *Pivot) Description() string {
	return description
}

func (p *Pivot) Apply(metrics ...cua.Metric) []cua.Metric {
	for _, m := range metrics {
		key, ok := m.GetTag(p.TagKey)
		if !ok {
			continue
		}

		value, ok := m.GetField(p.ValueKey)
		if !ok {
			continue
		}

		m.RemoveTag(p.TagKey)
		m.RemoveField(p.ValueKey)
		m.AddField(key, value)
	}
	return metrics
}

func init() {
	processors.Add("pivot", func() cua.Processor {
		return &Pivot{}
	})
}
