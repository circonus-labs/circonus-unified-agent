package parallel

import (
	"sync"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

type Unordered struct {
	wg      sync.WaitGroup
	acc     cua.Accumulator
	fn      func(cua.Metric) []cua.Metric
	inQueue chan cua.Metric
}

func NewUnordered(
	acc cua.Accumulator,
	fn func(cua.Metric) []cua.Metric,
	workerCount int,
) *Unordered {
	p := &Unordered{
		acc:     acc,
		inQueue: make(chan cua.Metric, workerCount),
		fn:      fn,
	}

	// start workers
	p.wg.Add(1)
	go func() {
		p.startWorkers(workerCount)
		p.wg.Done()
	}()

	return p
}

func (p *Unordered) startWorkers(count int) {
	wg := sync.WaitGroup{}
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			for metric := range p.inQueue {
				for _, m := range p.fn(metric) {
					p.acc.AddMetric(m)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func (p *Unordered) Stop() {
	close(p.inQueue)
	p.wg.Wait()
}

func (p *Unordered) Enqueue(m cua.Metric) {
	p.inQueue <- m
}
