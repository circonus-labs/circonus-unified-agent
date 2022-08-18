//go:build !linux && !freebsd
// +build !linux,!freebsd

package zfs

import (
	"context"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

func (z *Zfs) Gather(_ context.Context, _ cua.Accumulator) error {
	return nil
}

func init() {
	inputs.Add("zfs", func() cua.Input {
		return &Zfs{}
	})
}
