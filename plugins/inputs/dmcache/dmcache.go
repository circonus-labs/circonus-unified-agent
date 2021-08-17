package dmcache

import (
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type DMCache struct {
	getCurrentStatus func() ([]string, error)
	PerDevice        bool `toml:"per_device"`
}

var sampleConfig = `
  ## Whether to report per-device stats or not
  per_device = true
`

func (c *DMCache) SampleConfig() string {
	return sampleConfig
}

func (c *DMCache) Description() string {
	return "Provide a native collection for dmsetup based statistics for dm-cache"
}

func init() {
	inputs.Add("dmcache", func() cua.Input {
		return &DMCache{
			PerDevice:        true,
			getCurrentStatus: dmSetupStatus,
		}
	})
}
