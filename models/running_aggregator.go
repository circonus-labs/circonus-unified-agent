package models

import (
	"fmt"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/metric"
	"github.com/circonus-labs/circonus-unified-agent/selfstat"
)

type RunningAggregator struct {
	sync.Mutex
	Aggregator  cua.Aggregator
	Config      *AggregatorConfig
	periodStart time.Time
	periodEnd   time.Time
	log         cua.Logger

	MetricsPushed   selfstat.Stat
	MetricsFiltered selfstat.Stat
	MetricsDropped  selfstat.Stat
	PushTime        selfstat.Stat
}

func NewRunningAggregator(aggregator cua.Aggregator, config *AggregatorConfig) *RunningAggregator {
	tags := map[string]string{"aggregator": config.Name}
	if config.Alias != "" {
		tags["alias"] = config.Alias
	}

	aggErrorsRegister := selfstat.Register("aggregate", "errors", tags)
	logger := NewLogger("aggregators", config.Name, config.Alias)
	logger.OnErr(func() {
		aggErrorsRegister.Incr(1)
	})

	SetLoggerOnPlugin(aggregator, logger)

	return &RunningAggregator{
		Aggregator: aggregator,
		Config:     config,
		MetricsPushed: selfstat.Register(
			"aggregate",
			"metrics_pushed",
			tags,
		),
		MetricsFiltered: selfstat.Register(
			"aggregate",
			"metrics_filtered",
			tags,
		),
		MetricsDropped: selfstat.Register(
			"aggregate",
			"metrics_dropped",
			tags,
		),
		PushTime: selfstat.Register(
			"aggregate",
			"push_time_ns",
			tags,
		),
		log: logger,
	}
}

// AggregatorConfig is the common config for all aggregators.
type AggregatorConfig struct {
	Tags              map[string]string
	Name              string
	Alias             string
	MeasurementSuffix string
	NameOverride      string
	MeasurementPrefix string
	Filter            Filter
	Grace             time.Duration
	Period            time.Duration
	Delay             time.Duration
	DropOriginal      bool
}

func (r *RunningAggregator) LogName() string {
	return logName("aggregators", r.Config.Name, r.Config.Alias)
}

func (r *RunningAggregator) Init() error {
	if p, ok := r.Aggregator.(cua.Initializer); ok {
		err := p.Init()
		if err != nil {
			return fmt.Errorf("init (aggregator %s): %w", r.Config.Name, err)
		}
	}
	return nil
}

func (r *RunningAggregator) Period() time.Duration {
	return r.Config.Period
}

func (r *RunningAggregator) EndPeriod() time.Time {
	return r.periodEnd
}

func (r *RunningAggregator) UpdateWindow(start, until time.Time) {
	r.periodStart = start
	r.periodEnd = until
	r.log.Debugf("Updated aggregation range [%s, %s]", start, until)
}

func (r *RunningAggregator) MakeMetric(metric cua.Metric) cua.Metric {
	m := makemetric(
		metric,
		r.Config.NameOverride,
		r.Config.MeasurementPrefix,
		r.Config.MeasurementSuffix,
		r.Config.Tags,
		nil)

	if m != nil {
		m.SetAggregate(true)
	}

	r.MetricsPushed.Incr(1)

	return m
}

// Add a metric to the aggregator and return true if the original metric
// should be dropped.
func (r *RunningAggregator) Add(m cua.Metric) bool {
	if ok := r.Config.Filter.Select(m); !ok {
		return false
	}

	// Make a copy of the metric but don't retain tracking.  We do not fail a
	// delivery due to the aggregation not being sent because we can't create
	// aggregations of historical data.  Additionally, waiting for the
	// aggregation to be pushed would introduce a hefty latency to delivery.
	m = metric.FromMetric(m)

	r.Config.Filter.Modify(m)
	if len(m.FieldList()) == 0 {
		r.MetricsFiltered.Incr(1)
		return r.Config.DropOriginal
	}

	r.Lock()
	defer r.Unlock()

	if m.Time().Before(r.periodStart.Add(-r.Config.Grace)) || m.Time().After(r.periodEnd.Add(r.Config.Delay)) {
		r.log.Debugf("Metric is outside aggregation window; discarding. %s: m: %s e: %s g: %s",
			m.Time(), r.periodStart, r.periodEnd, r.Config.Grace)
		r.MetricsDropped.Incr(1)
		return r.Config.DropOriginal
	}

	r.Aggregator.Add(m)
	return r.Config.DropOriginal
}

func (r *RunningAggregator) Push(acc cua.Accumulator) {
	r.Lock()
	defer r.Unlock()

	since := r.periodEnd
	until := r.periodEnd.Add(r.Config.Period)
	r.UpdateWindow(since, until)

	r.push(acc)
	r.Aggregator.Reset()
}

func (r *RunningAggregator) push(acc cua.Accumulator) {
	start := time.Now()
	r.Aggregator.Push(acc)
	elapsed := time.Since(start)
	r.PushTime.Incr(elapsed.Nanoseconds())
}

func (r *RunningAggregator) Log() cua.Logger {
	return r.log
}
