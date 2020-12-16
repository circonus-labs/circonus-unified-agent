package inputs

import "github.com/circonus-labs/circonus-unified-agent/cua"

type Creator func() cua.Input

var Inputs = map[string]Creator{}

func Add(name string, creator Creator) {
	Inputs[name] = creator
}
