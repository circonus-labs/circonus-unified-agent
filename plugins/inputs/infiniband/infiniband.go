package infiniband

import "github.com/circonus-labs/circonus-unified-agent/cua"

// Stores the configuration values for the infiniband plugin - as there are no
// config values, this is intentionally empty
type Infiniband struct {
	Log cua.Logger `toml:"-"`
}

// Sample configuration for plugin
var InfinibandConfig = ``

func (_ *Infiniband) SampleConfig() string {
	return InfinibandConfig
}

func (_ *Infiniband) Description() string {
	return "Gets counters from all InfiniBand cards and ports installed"
}
