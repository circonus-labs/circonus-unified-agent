//go:build !linux
// +build !linux

package infiniband

import (
	"context"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

func (i *Infiniband) Init() error {
	i.Log.Warn("Current platform is not supported")
	return nil
}

func (*Infiniband) Gather(_ context.Context, _ cua.Accumulator) error {
	return nil
}

func init() {
	inputs.Add("infiniband", func() cua.Input {
		return &Infiniband{}
	})
}
