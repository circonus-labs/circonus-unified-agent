// +build windows

package processes

import (
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type Processes struct {
	Log cua.Logger
}

func (e *Processes) Init() error {
	e.Log.Warn("Current platform is not supported")
	return nil
}

func (e *Processes) Gather(acc cua.Accumulator) error {
	return nil
}

func init() {
	inputs.Add("processes", func() cua.Input {
		return &Processes{}
	})
}
