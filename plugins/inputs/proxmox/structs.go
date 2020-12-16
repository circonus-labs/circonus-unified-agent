package proxmox

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
)

type Proxmox struct {
	BaseURL         string            `toml:"base_url"`
	APIToken        string            `toml:"api_token"`
	ResponseTimeout internal.Duration `toml:"response_timeout"`
	NodeName        string            `toml:"node_name"`

	tls.ClientConfig

	httpClient       *http.Client
	nodeSearchDomain string

	requestFunction func(px *Proxmox, apiUrl string, method string, data url.Values) ([]byte, error)
	Log             cua.Logger `toml:"-"`
}

type VmCurrentStats struct {
	Data VmStat `json:"data"`
}

type ResourceType string

var (
	QEMU ResourceType = "qemu"
	LXC  ResourceType = "lxc"
)

type VmStats struct {
	Data []VmStat `json:"data"`
}

type VmStat struct {
	ID        string      `json:"vmid"`
	Name      string      `json:"name"`
	Status    string      `json:"status"`
	UsedMem   json.Number `json:"mem"`
	TotalMem  json.Number `json:"maxmem"`
	UsedDisk  json.Number `json:"disk"`
	TotalDisk json.Number `json:"maxdisk"`
	UsedSwap  json.Number `json:"swap"`
	TotalSwap json.Number `json:"maxswap"`
	Uptime    json.Number `json:"uptime"`
	CpuLoad   json.Number `json:"cpu"`
}

type VmConfig struct {
	Data struct {
		Searchdomain string `json:"searchdomain"`
		Hostname     string `json:"hostname"`
		Template     string `json:"template"`
	} `json:"data"`
}

type NodeDns struct {
	Data struct {
		Searchdomain string `json:"search"`
	} `json:"data"`
}
