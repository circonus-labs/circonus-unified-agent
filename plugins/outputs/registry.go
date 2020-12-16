package outputs

import "github.com/circonus-labs/circonus-unified-agent/cua"

type Creator func() cua.Output

var Outputs = map[string]Creator{}

func Add(name string, creator Creator) {
	Outputs[name] = creator
}
