package models

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/selfstat"
)

const (
	// Default size of metrics batch size.
	DefaultMetricBatchSize = 1000

	// Default number of metrics kept. It should be a multiple of batch size.
	DefaultMetricBufferLimit = 10000
)

// OutputConfig containing name and filter
type OutputConfig struct {
	Name              string
	Alias             string
	NamePrefix        string
	NameSuffix        string
	NameOverride      string
	Filter            Filter
	FlushJitter       time.Duration
	MetricBufferLimit int
	MetricBatchSize   int
	FlushInterval     time.Duration
}

// RunningOutput contains the output configuration
type RunningOutput struct {
	aggMutex          sync.Mutex
	MetricsFiltered   selfstat.Stat
	WriteTime         selfstat.Stat
	Output            cua.Output
	log               cua.Logger
	Config            *OutputConfig
	BatchReady        chan time.Time
	buffer            *Buffer
	newMetricsCount   int64
	droppedMetrics    int64
	MetricBufferLimit int
	MetricBatchSize   int
}

// original
// type RunningOutput struct {
// 	// Must be 64-bit aligned
// 	newMetricsCount int64
// 	droppedMetrics  int64

// 	Output            cua.Output
// 	Config            *OutputConfig
// 	MetricBufferLimit int
// 	MetricBatchSize   int

// 	MetricsFiltered selfstat.Stat
// 	WriteTime       selfstat.Stat

// 	BatchReady chan time.Time

// 	buffer *Buffer
// 	log    cua.Logger

// 	aggMutex sync.Mutex
// }

func NewRunningOutput(
	name string,
	output cua.Output,
	config *OutputConfig,
	batchSize int,
	bufferLimit int,
) *RunningOutput {
	tags := map[string]string{"output": config.Name}
	if config.Alias != "" {
		tags["alias"] = config.Alias
	}

	writeErrorsRegister := selfstat.Register("write", "errors", tags)
	logger := NewLogger("outputs", config.Name, config.Alias)
	logger.OnErr(func() {
		writeErrorsRegister.Incr(1)
	})
	SetLoggerOnPlugin(output, logger)

	if config.MetricBufferLimit > 0 {
		bufferLimit = config.MetricBufferLimit
	}
	if bufferLimit == 0 {
		bufferLimit = DefaultMetricBufferLimit
	}
	if config.MetricBatchSize > 0 {
		batchSize = config.MetricBatchSize
	}
	if batchSize == 0 {
		batchSize = DefaultMetricBatchSize
	}

	ro := &RunningOutput{
		buffer:            NewBuffer(config.Name, config.Alias, bufferLimit),
		BatchReady:        make(chan time.Time, 1),
		Output:            output,
		Config:            config,
		MetricBufferLimit: bufferLimit,
		MetricBatchSize:   batchSize,
		MetricsFiltered: selfstat.Register(
			"write",
			"metrics_filtered",
			tags,
		),
		WriteTime: selfstat.RegisterTiming(
			"write",
			"write_time_ns",
			tags,
		),
		log: logger,
	}

	return ro
}

func (ro *RunningOutput) LogName() string {
	return logName("outputs", ro.Config.Name, ro.Config.Alias)
}

func (ro *RunningOutput) metricFiltered(metric cua.Metric) {
	ro.MetricsFiltered.Incr(1)
	metric.Drop()
}

func (ro *RunningOutput) Init() error {
	if p, ok := ro.Output.(cua.Initializer); ok {
		err := p.Init()
		if err != nil {
			return fmt.Errorf("init (output %s): %w", ro.Config.Name, err)
		}

	}
	return nil
}

// AddMetric adds a metric to the output.
//
// Takes ownership of metric
func (ro *RunningOutput) AddMetric(metric cua.Metric) {
	if ok := ro.Config.Filter.Select(metric); !ok {
		ro.metricFiltered(metric)
		return
	}

	ro.Config.Filter.Modify(metric)
	if len(metric.FieldList()) == 0 {
		ro.metricFiltered(metric)
		return
	}

	if output, ok := ro.Output.(cua.AggregatingOutput); ok {
		ro.aggMutex.Lock()
		output.Add(metric)
		ro.aggMutex.Unlock()
		return
	}

	if len(ro.Config.NameOverride) > 0 {
		metric.SetName(ro.Config.NameOverride)
	}

	if len(ro.Config.NamePrefix) > 0 {
		metric.AddPrefix(ro.Config.NamePrefix)
	}

	if len(ro.Config.NameSuffix) > 0 {
		metric.AddSuffix(ro.Config.NameSuffix)
	}

	dropped := ro.buffer.Add(metric)
	atomic.AddInt64(&ro.droppedMetrics, int64(dropped))

	count := atomic.AddInt64(&ro.newMetricsCount, 1)
	if count == int64(ro.MetricBatchSize) {
		atomic.StoreInt64(&ro.newMetricsCount, 0)
		select {
		case ro.BatchReady <- time.Now():
		default:
		}
	}
}

// Write writes all metrics to the output, stopping when all have been sent on
// or error.
func (ro *RunningOutput) Write() error {
	if output, ok := ro.Output.(cua.AggregatingOutput); ok {
		ro.aggMutex.Lock()
		metrics := output.Push()
		ro.buffer.Add(metrics...)
		output.Reset()
		ro.aggMutex.Unlock()
	}

	atomic.StoreInt64(&ro.newMetricsCount, 0)

	// Only process the metrics in the buffer now.  Metrics added while we are
	// writing will be sent on the next call.
	nBuffer := ro.buffer.Len()
	nBatches := nBuffer/ro.MetricBatchSize + 1
	for i := 0; i < nBatches; i++ {
		batch := ro.buffer.Batch(ro.MetricBatchSize)
		if len(batch) == 0 {
			break
		}

		err := ro.write(batch)
		if err != nil {
			ro.buffer.Reject(batch)
			return err
		}
		ro.buffer.Accept(batch)
	}
	return nil
}

// WriteBatch writes a single batch of metrics to the output.
func (ro *RunningOutput) WriteBatch() error {
	batch := ro.buffer.Batch(ro.MetricBatchSize)
	if len(batch) == 0 {
		return nil
	}

	err := ro.write(batch)
	if err != nil {
		ro.buffer.Reject(batch)
		return err
	}
	ro.buffer.Accept(batch)

	return nil
}

// Close closes the output
func (ro *RunningOutput) Close() {
	err := ro.Output.Close()
	if err != nil {
		ro.log.Errorf("Error closing output: %v", err)
	}
}

func (ro *RunningOutput) write(metrics []cua.Metric) error {
	dropped := atomic.LoadInt64(&ro.droppedMetrics)
	if dropped > 0 {
		ro.log.Warnf("Metric buffer overflow; %d batches have been dropped", dropped)
		atomic.StoreInt64(&ro.droppedMetrics, 0)
	}

	start := time.Now()
	_, err := ro.Output.Write(metrics)
	elapsed := time.Since(start)
	ro.WriteTime.Incr(elapsed.Nanoseconds())

	if err == nil {
		ro.log.Debugf("Wrote %d batches in %s", len(metrics), elapsed)
	}
	if err != nil {
		return fmt.Errorf("write (output %s): %w", ro.Config.Name, err)
	}
	return nil
}

func (ro *RunningOutput) LogBufferStatus() {
	nBuffer := ro.buffer.Len()
	ro.log.Debugf("Buffer fullness: %d / %d batches", nBuffer, ro.MetricBufferLimit)
}

func (ro *RunningOutput) Log() cua.Logger {
	return ro.log
}

func (ro *RunningOutput) BufferLength() int {
	return ro.buffer.Len()
}
