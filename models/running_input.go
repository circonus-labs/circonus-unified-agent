package models

import (
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/selfstat"
)

var (
	GlobalMetricsGathered = selfstat.Register("agent", "metrics_gathered", map[string]string{})
	GlobalGatherErrors    = selfstat.Register("agent", "gather_errors", map[string]string{})
)

type RunningInput struct {
	Input  cua.Input
	Config *InputConfig

	log         cua.Logger
	defaultTags map[string]string

	MetricsGathered selfstat.Stat
	GatherTime      selfstat.Stat
}

func NewRunningInput(input cua.Input, config *InputConfig) *RunningInput {
	tags := map[string]string{"input": config.Name}
	if config.Alias != "" {
		tags["alias"] = config.Alias
	}

	inputErrorsRegister := selfstat.Register("gather", "errors", tags)
	logger := NewLogger("inputs", config.Name, config.Alias)
	logger.OnErr(func() {
		inputErrorsRegister.Incr(1)
		GlobalGatherErrors.Incr(1)
	})
	SetLoggerOnPlugin(input, logger)

	return &RunningInput{
		Input:  input,
		Config: config,
		MetricsGathered: selfstat.Register(
			"gather",
			"metrics_gathered",
			tags,
		),
		GatherTime: selfstat.RegisterTiming(
			"gather",
			"gather_time_ns",
			tags,
		),
		log: logger,
	}
}

// InputConfig is the common config for all inputs.
type InputConfig struct {
	Name             string
	InstanceID       string
	Alias            string
	Interval         time.Duration
	CollectionJitter time.Duration
	Precision        time.Duration

	NameOverride      string
	MeasurementPrefix string
	MeasurementSuffix string
	Tags              map[string]string
	Filter            Filter
}

func (r *RunningInput) metricFiltered(metric cua.Metric) {
	metric.Drop()
}

func (r *RunningInput) LogName() string {
	return logName("inputs", r.Config.Name, r.Config.Alias)
}

func (r *RunningInput) Init() error {
	if p, ok := r.Input.(cua.Initializer); ok {
		err := p.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RunningInput) MakeMetric(metric cua.Metric) cua.Metric {
	if ok := r.Config.Filter.Select(metric); !ok {
		r.metricFiltered(metric)
		return nil
	}

	m := makemetric(
		metric,
		r.Config.NameOverride,
		r.Config.MeasurementPrefix,
		r.Config.MeasurementSuffix,
		r.Config.Tags,
		r.defaultTags)

	m.SetOrigin(r.Config.Name)
	m.SetOriginInstance(r.Config.InstanceID)

	r.Config.Filter.Modify(metric)
	if len(metric.FieldList()) == 0 {
		r.metricFiltered(metric)
		return nil
	}

	r.MetricsGathered.Incr(1)
	GlobalMetricsGathered.Incr(1)
	return m
}

func (r *RunningInput) Gather(acc cua.Accumulator) error {
	start := time.Now()
	err := r.Input.Gather(acc)
	elapsed := time.Since(start)
	r.GatherTime.Incr(elapsed.Nanoseconds())
	return err
}

func (r *RunningInput) SetDefaultTags(tags map[string]string) {
	r.defaultTags = tags
}

func (r *RunningInput) Log() cua.Logger {
	return r.log
}
