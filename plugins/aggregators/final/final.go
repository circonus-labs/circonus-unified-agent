package final

import (
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/aggregators"
)

var sampleConfig = `
  ## The period on which to flush & clear the aggregator.
  period = "30s"
  ## If true, the original metric will be dropped by the
  ## aggregator and will not get sent to the output plugins.
  drop_original = false

  ## The time that a series is not updated until considering it final.
  series_timeout = "5m"
`

type Final struct {
	SeriesTimeout internal.Duration `toml:"series_timeout"`

	// The last metric for all series which are active
	metricCache map[uint64]cua.Metric
}

func NewFinal() *Final {
	return &Final{
		SeriesTimeout: internal.Duration{Duration: 5 * time.Minute},
		metricCache:   make(map[uint64]cua.Metric),
	}
}

func (m *Final) SampleConfig() string {
	return sampleConfig
}

func (m *Final) Description() string {
	return "Report the final metric of a series"
}

func (m *Final) Add(in cua.Metric) {
	id := in.HashID()
	m.metricCache[id] = in
}

func (m *Final) Push(acc cua.Accumulator) {
	// Preserve timestamp of original metric
	acc.SetPrecision(time.Nanosecond)

	for id, metric := range m.metricCache {
		if time.Since(metric.Time()) > m.SeriesTimeout.Duration {
			fields := map[string]interface{}{}
			for _, field := range metric.FieldList() {
				fields[field.Key+"_final"] = field.Value
			}
			acc.AddFields(metric.Name(), fields, metric.Tags(), metric.Time())
			delete(m.metricCache, id)
		}
	}
}

func (m *Final) Reset() {
}

func init() {
	aggregators.Add("final", func() cua.Aggregator {
		return NewFinal()
	})
}
