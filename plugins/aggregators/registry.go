package aggregators

import "github.com/circonus-labs/circonus-unified-agent/cua"

type Creator func() cua.Aggregator

var Aggregators = map[string]Creator{}

func Add(name string, creator Creator) {
	Aggregators[name] = creator
}
