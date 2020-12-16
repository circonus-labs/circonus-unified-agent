package wireless

import (
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

// Wireless is used to store configuration values.
type Wireless struct {
	HostProc string     `toml:"host_proc"`
	Log      cua.Logger `toml:"-"`
}

var sampleConfig = `
  ## Sets 'proc' directory path
  ## If not specified, then default is /proc
  # host_proc = "/proc"
`

// Description returns information about the plugin.
func (w *Wireless) Description() string {
	return "Monitor wifi signal strength and quality"
}

// SampleConfig displays configuration instructions.
func (w *Wireless) SampleConfig() string {
	return sampleConfig
}

func init() {
	inputs.Add("wireless", func() cua.Input {
		return &Wireless{}
	})
}
