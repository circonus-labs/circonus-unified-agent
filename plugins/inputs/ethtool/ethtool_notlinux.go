//go:build !linux
// +build !linux

package ethtool

import (
	"context"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

func (e *Ethtool) Init() error {
	e.Log.Warn("Current platform is not supported")
	return nil
}

func (e *Ethtool) Gather(_ context.Context, _ cua.Accumulator) error {
	return nil
}

func init() {
	inputs.Add(pluginName, func() cua.Input {
		return &Ethtool{command: nil}
	})
}
