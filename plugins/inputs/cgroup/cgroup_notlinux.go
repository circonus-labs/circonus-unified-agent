//go:build !linux
// +build !linux

package cgroup

import (
	"context"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

func (g *CGroup) Gather(_ context.Context, _ cua.Accumulator) error {
	return nil
}
