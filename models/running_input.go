package models

import (
	"context"
	"fmt"
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
	if config.InstanceID != "" {
		tags["instance_id"] = config.InstanceID
	}

	alias := config.Alias
	if alias == "" && config.InstanceID != "" {
		alias = config.InstanceID
	}

	inputErrorsRegister := selfstat.Register("gather", "errors", tags)
	logger := NewLogger("inputs", config.Name, alias)
	logger.OnErr(func() {
		inputErrorsRegister.Incr(1)
		GlobalGatherErrors.Incr(1)
	})
	SetLoggerOnPlugin(input, logger)
	// add for high performance (hp) plugins SetInstanceIDOnPlugin(input,config.InstanceID)

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
	Tags              map[string]string
	Name              string
	InstanceID        string
	Alias             string
	NameOverride      string
	MeasurementPrefix string
	MeasurementSuffix string
	Filter            Filter
	Precision         time.Duration
	Interval          time.Duration
	CollectionJitter  time.Duration
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
			return fmt.Errorf("init (input %s): %w", r.Config.Name, err)
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

func (r *RunningInput) Gather(ctx context.Context, acc cua.Accumulator) error {
	start := time.Now()
	err := r.Input.Gather(ctx, acc)
	elapsed := time.Since(start)
	r.GatherTime.Incr(elapsed.Nanoseconds())
	if err != nil {
		return fmt.Errorf("gather (input %s): %w", r.Config.Name, err)
	}
	return nil
}

func (r *RunningInput) SetDefaultTags(tags map[string]string) {
	r.defaultTags = tags
}

func (r *RunningInput) Log() cua.Logger {
	return r.log
}
