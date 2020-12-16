package seriesgrouper

import (
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/metric"
	"github.com/circonus-labs/circonus-unified-agent/plugins/aggregators"
)

const (
	description  = "Merge metrics into multifield metrics by series key"
	sampleConfig = `
  ## If true, the original metric will be dropped by the
  ## aggregator and will not get sent to the output plugins.
  drop_original = true
`
)

type Merge struct {
	grouper *metric.SeriesGrouper
	log     cua.Logger
}

func (a *Merge) Init() error {
	a.grouper = metric.NewSeriesGrouper()
	return nil
}

func (a *Merge) Description() string {
	return description
}

func (a *Merge) SampleConfig() string {
	return sampleConfig
}

func (a *Merge) Add(m cua.Metric) {
	tags := m.Tags()
	for _, field := range m.FieldList() {
		err := a.grouper.Add(m.Name(), tags, m.Time(), field.Key, field.Value)
		if err != nil {
			a.log.Errorf("Error adding metric: %v", err)
		}
	}
}

func (a *Merge) Push(acc cua.Accumulator) {
	// Always use nanosecond precision to avoid rounding metrics that were
	// produced at a precision higher than the agent default.
	acc.SetPrecision(time.Nanosecond)

	for _, m := range a.grouper.Metrics() {
		acc.AddMetric(m)
	}
}

func (a *Merge) Reset() {
	a.grouper = metric.NewSeriesGrouper()
}

func init() {
	aggregators.Add("merge", func() cua.Aggregator {
		return &Merge{}
	})
}
