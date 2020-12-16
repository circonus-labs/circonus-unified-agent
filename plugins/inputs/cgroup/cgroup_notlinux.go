// +build !linux

package cgroup

import "github.com/circonus-labs/circonus-unified-agent/cua"

func (g *CGroup) Gather(acc cua.Accumulator) error {
	return nil
}
