package models

import (
	"fmt"
	"sync"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/selfstat"
)

type RunningProcessor struct {
	sync.Mutex
	log       cua.Logger
	Processor cua.StreamingProcessor
	Config    *ProcessorConfig
}

type RunningProcessors []*RunningProcessor

func (rp RunningProcessors) Len() int           { return len(rp) }
func (rp RunningProcessors) Swap(i, j int)      { rp[i], rp[j] = rp[j], rp[i] }
func (rp RunningProcessors) Less(i, j int) bool { return rp[i].Config.Order < rp[j].Config.Order }

// FilterConfig containing a name and filter
type ProcessorConfig struct {
	Name   string
	Alias  string
	Order  int64
	Filter Filter
}

func NewRunningProcessor(processor cua.StreamingProcessor, config *ProcessorConfig) *RunningProcessor {
	tags := map[string]string{"processor": config.Name}
	if config.Alias != "" {
		tags["alias"] = config.Alias
	}

	processErrorsRegister := selfstat.Register("process", "errors", tags)
	logger := NewLogger("processors", config.Name, config.Alias)
	logger.OnErr(func() {
		processErrorsRegister.Incr(1)
	})
	SetLoggerOnPlugin(processor, logger)

	return &RunningProcessor{
		Processor: processor,
		Config:    config,
		log:       logger,
	}
}

func (rp *RunningProcessor) metricFiltered(metric cua.Metric) {
	metric.Drop()
}

func (rp *RunningProcessor) Init() error {
	if p, ok := rp.Processor.(cua.Initializer); ok {
		err := p.Init()
		if err != nil {
			return fmt.Errorf("init (processor %s): %w", rp.Config.Name, err)
		}
	}
	return nil
}

func (rp *RunningProcessor) Log() cua.Logger {
	return rp.log
}

func (rp *RunningProcessor) LogName() string {
	return logName("processors", rp.Config.Name, rp.Config.Alias)
}

func (rp *RunningProcessor) MakeMetric(metric cua.Metric) cua.Metric {
	return metric
}

func (rp *RunningProcessor) Start(acc cua.Accumulator) error {
	return rp.Processor.Start(acc)
}

func (rp *RunningProcessor) Add(m cua.Metric, acc cua.Accumulator) error {
	if ok := rp.Config.Filter.Select(m); !ok {
		// pass downstream
		acc.AddMetric(m)
		return nil
	}

	rp.Config.Filter.Modify(m)
	if len(m.FieldList()) == 0 {
		// drop metric
		rp.metricFiltered(m)
		return nil
	}

	return rp.Processor.Add(m, acc)
}

func (rp *RunningProcessor) Stop() {
	_ = rp.Processor.Stop()
}
