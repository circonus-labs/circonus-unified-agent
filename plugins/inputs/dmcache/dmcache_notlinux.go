//go:build !linux
// +build !linux

package dmcache

import (
	"context"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

func (c *DMCache) Gather(_ context.Context, _ cua.Accumulator) error {
	return nil
}

func dmSetupStatus() ([]string, error) {
	return []string{}, nil
}
