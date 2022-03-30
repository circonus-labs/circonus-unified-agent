package infiniband

import "github.com/circonus-labs/circonus-unified-agent/cua"

// Stores the configuration values for the infiniband plugin - as there are no
// config values, this is intentionally empty
type Infiniband struct {
	Log cua.Logger `toml:"-"`
}

var InfinibandConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)
`

// SampleConfig example configuration for plugin
func (*Infiniband) SampleConfig() string {
	return InfinibandConfig
}

func (*Infiniband) Description() string {
	return "Gets counters from all InfiniBand cards and ports installed"
}
