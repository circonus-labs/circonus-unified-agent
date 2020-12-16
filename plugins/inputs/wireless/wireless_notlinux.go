// +build !linux

package wireless

import (
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

func (w *Wireless) Init() error {
	w.Log.Warn("Current platform is not supported")
	return nil
}

func (w *Wireless) Gather(acc cua.Accumulator) error {
	return nil
}

func init() {
	inputs.Add("wireless", func() cua.Input {
		return &Wireless{}
	})
}
