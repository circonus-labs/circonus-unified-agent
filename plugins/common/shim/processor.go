package shim

import (
	"bufio"
	"fmt"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/agent"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/parsers"
	"github.com/circonus-labs/circonus-unified-agent/plugins/processors"
)

// AddProcessor adds the processor to the shim. Later calls to Run() will run this.
func (s *Shim) AddProcessor(processor cua.Processor) error {
	setLoggerOnPlugin(processor, s.Log())
	p := processors.NewStreamingProcessorFromProcessor(processor)
	return s.AddStreamingProcessor(p)
}

// AddStreamingProcessor adds the processor to the shim. Later calls to Run() will run this.
func (s *Shim) AddStreamingProcessor(processor cua.StreamingProcessor) error {
	setLoggerOnPlugin(processor, s.Log())
	if p, ok := processor.(cua.Initializer); ok {
		err := p.Init()
		if err != nil {
			return fmt.Errorf("failed to init input: %w", err)
		}
	}

	s.Processor = processor
	return nil
}

func (s *Shim) RunProcessor() error {
	acc := agent.NewAccumulator(s, s.metricCh)
	acc.SetPrecision(time.Nanosecond)

	parser, err := parsers.NewInfluxParser()
	if err != nil {
		return fmt.Errorf("Failed to create new parser: %w", err)
	}

	err = s.Processor.Start(acc)
	if err != nil {
		return fmt.Errorf("failed to start processor: %w", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		_ = s.writeProcessedMetrics()
		wg.Done()
	}()

	scanner := bufio.NewScanner(s.stdin)
	for scanner.Scan() {
		m, err := parser.ParseLine(scanner.Text())
		if err != nil {
			fmt.Fprintf(s.stderr, "Failed to parse metric: %s\b", err)
			continue
		}
		_ = s.Processor.Add(m, acc)
	}

	close(s.metricCh)
	_ = s.Processor.Stop()
	wg.Wait()
	return nil
}
