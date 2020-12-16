// +build !linux

package synproxy

import (
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

func (k *Synproxy) Init() error {
	k.Log.Warn("Current platform is not supported")
	return nil
}

func (k *Synproxy) Gather(acc cua.Accumulator) error {
	return nil
}

func init() {
	inputs.Add("synproxy", func() cua.Input {
		return &Synproxy{}
	})
}
