package synproxy

import (
	"log"
	"os"
	"path"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type Synproxy struct {
	Log cua.Logger `toml:"-"`

	// Synproxy stats filename (proc filesystem)
	statFile string
}

func (k *Synproxy) Description() string {
	return "Get synproxy counter statistics from procfs"
}

func (k *Synproxy) SampleConfig() string {
	return `
  instance_id = "" # unique instance identifier (REQUIRED)
`
}

func getHostProc() string {
	procPath := "/proc"
	if os.Getenv("HOST_PROC") != "" {
		procPath = os.Getenv("HOST_PROC")
	}
	log.Print("I! Using default procPath: " + procPath)
	return procPath
}

func init() {
	inputs.Add("synproxy", func() cua.Input {
		return &Synproxy{
			statFile: path.Join(getHostProc(), "/net/stat/synproxy"),
		}
	})
}
