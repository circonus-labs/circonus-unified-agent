// +build !linux,!freebsd

package zfs

import (
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

func (z *Zfs) Gather(acc cua.Accumulator) error {
	return nil
}

func init() {
	inputs.Add("zfs", func() cua.Input {
		return &Zfs{}
	})
}
