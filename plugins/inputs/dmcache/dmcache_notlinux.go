// +build !linux

package dmcache

import "github.com/circonus-labs/circonus-unified-agent/cua"

func (c *DMCache) Gather(acc cua.Accumulator) error {
	return nil
}

func dmSetupStatus() ([]string, error) {
	return []string{}, nil
}
