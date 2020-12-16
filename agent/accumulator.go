package agent

import (
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/metric"
)

type MetricMaker interface {
	LogName() string
	MakeMetric(metric cua.Metric) cua.Metric
	Log() cua.Logger
}

type accumulator struct {
	maker     MetricMaker
	metrics   chan<- cua.Metric
	precision time.Duration
}

func NewAccumulator(
	maker MetricMaker,
	metrics chan<- cua.Metric,
) cua.Accumulator {
	acc := accumulator{
		maker:     maker,
		metrics:   metrics,
		precision: time.Nanosecond,
	}
	return &acc
}

func (ac *accumulator) AddFields(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	t ...time.Time,
) {
	ac.addFields(measurement, tags, fields, cua.Untyped, t...)
}

func (ac *accumulator) AddGauge(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	t ...time.Time,
) {
	ac.addFields(measurement, tags, fields, cua.Gauge, t...)
}

func (ac *accumulator) AddCounter(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	t ...time.Time,
) {
	ac.addFields(measurement, tags, fields, cua.Counter, t...)
}

func (ac *accumulator) AddSummary(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	t ...time.Time,
) {
	ac.addFields(measurement, tags, fields, cua.Summary, t...)
}

func (ac *accumulator) AddHistogram(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	t ...time.Time,
) {
	ac.addFields(measurement, tags, fields, cua.Histogram, t...)
}

func (ac *accumulator) AddMetric(m cua.Metric) {
	m.SetTime(m.Time().Round(ac.precision))
	if m := ac.maker.MakeMetric(m); m != nil {
		ac.metrics <- m
	}
}

func (ac *accumulator) addFields(
	measurement string,
	tags map[string]string,
	fields map[string]interface{},
	tp cua.ValueType,
	t ...time.Time,
) {
	m, err := metric.New(measurement, tags, fields, ac.getTime(t), tp)
	if err != nil {
		return
	}
	if m := ac.maker.MakeMetric(m); m != nil {
		ac.metrics <- m
	}
}

// AddError passes a runtime error to the accumulator.
// The error will be tagged with the plugin name and written to the log.
func (ac *accumulator) AddError(err error) {
	if err == nil {
		return
	}
	ac.maker.Log().Errorf("Error in plugin: %v", err)
}

func (ac *accumulator) SetPrecision(precision time.Duration) {
	ac.precision = precision
}

func (ac *accumulator) getTime(t []time.Time) time.Time {
	var timestamp time.Time
	if len(t) > 0 {
		timestamp = t[0]
	} else {
		timestamp = time.Now()
	}
	return timestamp.Round(ac.precision)
}

func (ac *accumulator) WithTracking(maxTracked int) cua.TrackingAccumulator {
	return &trackingAccumulator{
		Accumulator: ac,
		delivered:   make(chan cua.DeliveryInfo, maxTracked),
	}
}

type trackingAccumulator struct {
	cua.Accumulator
	delivered chan cua.DeliveryInfo
}

func (a *trackingAccumulator) AddTrackingMetric(m cua.Metric) cua.TrackingID {
	dm, id := metric.WithTracking(m, a.onDelivery)
	a.AddMetric(dm)
	return id
}

func (a *trackingAccumulator) AddTrackingMetricGroup(group []cua.Metric) cua.TrackingID {
	db, id := metric.WithGroupTracking(group, a.onDelivery)
	for _, m := range db {
		a.AddMetric(m)
	}
	return id
}

func (a *trackingAccumulator) Delivered() <-chan cua.DeliveryInfo {
	return a.delivered
}

func (a *trackingAccumulator) onDelivery(info cua.DeliveryInfo) {
	select {
	case a.delivered <- info:
	default:
		// This is a programming error in the input.  More items were sent for
		// tracking than space requested.
		panic("channel is full")
	}
}
