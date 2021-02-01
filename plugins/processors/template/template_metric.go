package template

import (
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

type Metric struct {
	metric cua.Metric
}

func (m *Metric) Name() string {
	return m.metric.Name()
}

func (m *Metric) Tag(key string) string {
	tagString, _ := m.metric.GetTag(key)
	return tagString
}

func (m *Metric) Field(key string) interface{} {
	field, _ := m.metric.GetField(key)
	return field
}

func (m *Metric) Time() time.Time {
	return m.metric.Time()
}
