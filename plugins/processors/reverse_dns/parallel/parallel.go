package parallel

import "github.com/circonus-labs/circonus-unified-agent/cua"

type Parallel interface {
	Enqueue(cua.Metric)
	Stop()
}
