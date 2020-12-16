package template

import (
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

type TemplateMetric struct {
	metric cua.Metric
}

func (m *TemplateMetric) Name() string {
	return m.metric.Name()
}

func (m *TemplateMetric) Tag(key string) string {
	tagString, _ := m.metric.GetTag(key)
	return tagString
}

func (m *TemplateMetric) Field(key string) interface{} {
	field, _ := m.metric.GetField(key)
	return field
}

func (m *TemplateMetric) Time() time.Time {
	return m.metric.Time()
}
