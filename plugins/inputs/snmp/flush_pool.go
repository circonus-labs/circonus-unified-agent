package snmp

import (
	"bytes"
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/go-trapmetrics"
)

var flusherPool flusher
var flusherPoolmu sync.Mutex

type flusher struct {
	wg          sync.WaitGroup
	traps       chan trap
	log         cua.Logger
	initialized bool
}

type trap struct {
	name string
	ctx  context.Context
	dest *trapmetrics.TrapMetrics
	tags trapmetrics.Tags
}

func initFlusherPool(logger cua.Logger, poolSize, queueSize uint) {
	flusherPoolmu.Lock()
	defer flusherPoolmu.Unlock()

	if flusherPool.initialized {
		return
	}

	flusherQueueSize := uint(100)
	if queueSize > 0 {
		flusherQueueSize = queueSize
	}

	flusherPool = flusher{
		log:         logger,
		traps:       make(chan trap, flusherQueueSize),
		initialized: true,
	}

	flusherPoolSize := 4
	np := runtime.NumCPU()
	switch {
	case np > 10:
		flusherPoolSize = np - 2
	case np > 4:
		flusherPoolSize = np - 1
	}
	// override with user setting
	if poolSize > 0 {
		flusherPoolSize = int(poolSize)
	}

	flusherPool.wg.Add(flusherPoolSize)
	flusherPool.log.Debugf("starting %d metric flushers", flusherPoolSize)
	for i := 1; i <= flusherPoolSize; i++ {
		i := i
		go func(id int) {
			var buf bytes.Buffer
			// buf.Grow(32768)
			for t := range flusherPool.traps {
				buf.Reset()
				fstart := time.Now()
				if _, err := t.dest.FlushWithBuffer(t.ctx, buf); err != nil {
					flusherPool.log.Warnf("flusher %d: %s", id, err)
				}
				_ = t.dest.GaugeSet("dur_last_submit", t.tags, time.Since(fstart).Seconds(), nil)
				if done(t.ctx) {
					break
				}
			}
			flusherPool.wg.Done()
		}(i)
	}
}

func done(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
