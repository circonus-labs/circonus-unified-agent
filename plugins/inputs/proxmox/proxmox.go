package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

var sampleConfig = `
  ## API connection configuration. The API token was introduced in Proxmox v6.2. Required permissions for user and token: PVEAuditor role on /.
  base_url = "https://localhost:8006/api2/json"
  api_token = "USER@REALM!TOKENID=UUID"
  ## Node name, defaults to OS hostname
  # node_name = ""

  ## Optional TLS Config
  # tls_ca = "/etc/circonus-unified-agent/ca.pem"
  # tls_cert = "/etc/circonus-unified-agent/cert.pem"
  # tls_key = "/etc/circonus-unified-agent/key.pem"
  ## Use TLS but skip chain & host verification
  insecure_skip_verify = false

  # HTTP response timeout (default: 5s)
  response_timeout = "5s"
`

func (px *Proxmox) SampleConfig() string {
	return sampleConfig
}

func (px *Proxmox) Description() string {
	return "Provides metrics from Proxmox nodes (Proxmox Virtual Environment > 6.2)."
}

func (px *Proxmox) Gather(ctx context.Context, acc cua.Accumulator) error {
	err := getNodeSearchDomain(px)
	if err != nil {
		return err
	}

	gatherLxcData(px, acc)
	gatherQemuData(px, acc)

	return nil
}

func (px *Proxmox) Init() error {
	if px.NodeName == "" {
		hostname, _ := os.Hostname()
		px.NodeName = hostname
	}

	tlsCfg, err := px.ClientConfig.TLSConfig()
	if err != nil {
		return fmt.Errorf("TLSConfig: %w", err)
	}
	px.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
		Timeout: px.ResponseTimeout.Duration,
	}

	return nil
}

func init() {
	inputs.Add("proxmox", func() cua.Input {
		return &Proxmox{
			requestFunction: performRequest,
		}
	})
}

func getNodeSearchDomain(px *Proxmox) error {
	apiURL := "/nodes/" + px.NodeName + "/dns"
	jsonData, err := px.requestFunction(px, apiURL, http.MethodGet, nil)
	if err != nil {
		return err
	}

	var nodeDNS NodeDNS
	err = json.Unmarshal(jsonData, &nodeDNS)
	if err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	if nodeDNS.Data.Searchdomain == "" {
		return fmt.Errorf("search domain is not set")
	}
	px.nodeSearchDomain = nodeDNS.Data.Searchdomain

	return nil
}

func performRequest(px *Proxmox, apiURL string, method string, data url.Values) ([]byte, error) {
	request, err := http.NewRequest(method, px.BaseURL+apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("http new req (%s): %w", px.BaseURL+apiURL, err)
	}
	request.Header.Add("Authorization", "PVEAPIToken="+px.APIToken)

	resp, err := px.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("readall: %w", err)
	}

	return responseBody, nil
}

func gatherLxcData(px *Proxmox, acc cua.Accumulator) {
	gatherVMData(px, acc, LXC)
}

func gatherQemuData(px *Proxmox, acc cua.Accumulator) {
	gatherVMData(px, acc, QEMU)
}

func gatherVMData(px *Proxmox, acc cua.Accumulator, rt ResourceType) {
	vmStats, err := getVMStats(px, rt)
	if err != nil {
		px.Log.Errorf("error getting VM stats: %v", err)
		return
	}

	// For each VM add metrics to Accumulator
	for _, vmStat := range vmStats.Data {
		vmConfig, err := getVMConfig(px, vmStat.ID, rt)
		if err != nil {
			px.Log.Errorf("error getting VM config: %v", err)
			return
		}
		tags := getTags(px, vmStat.Name, vmConfig, rt)

		currentVMStatus, err := getCurrentVMStatus(px, rt, vmStat.ID)
		if err != nil {
			px.Log.Errorf("error getting current VM status: %v", err)
			return
		}

		fields := getFields(currentVMStatus)
		// getFields always returns nil for err
		// if err != nil {
		// 	px.Log.Errorf("error getting VM measurements: %v", err)
		// 	return
		// }
		acc.AddFields("proxmox", fields, tags)
	}
}

func getCurrentVMStatus(px *Proxmox, rt ResourceType, id string) (VMStat, error) {
	apiURL := "/nodes/" + px.NodeName + "/" + string(rt) + "/" + id + "/status/current"

	jsonData, err := px.requestFunction(px, apiURL, http.MethodGet, nil)
	if err != nil {
		return VMStat{}, err
	}

	var currentVMStatus VMCurrentStats
	if err := json.Unmarshal(jsonData, &currentVMStatus); err != nil {
		return VMStat{}, fmt.Errorf("json unmarshal: %w", err)
	}

	return currentVMStatus.Data, nil
}

func getVMStats(px *Proxmox, rt ResourceType) (VMStats, error) {
	apiURL := "/nodes/" + px.NodeName + "/" + string(rt)
	jsonData, err := px.requestFunction(px, apiURL, http.MethodGet, nil)
	if err != nil {
		return VMStats{}, err
	}

	var vmStats VMStats
	err = json.Unmarshal(jsonData, &vmStats)
	if err != nil {
		return VMStats{}, fmt.Errorf("json unmarshal: %w", err)
	}

	return vmStats, nil
}

func getVMConfig(px *Proxmox, vmID string, rt ResourceType) (VMConfig, error) {
	apiURL := "/nodes/" + px.NodeName + "/" + string(rt) + "/" + vmID + "/config"
	jsonData, err := px.requestFunction(px, apiURL, http.MethodGet, nil)
	if err != nil {
		return VMConfig{}, err
	}

	var vmConfig VMConfig
	err = json.Unmarshal(jsonData, &vmConfig)
	if err != nil {
		return VMConfig{}, fmt.Errorf("json unmarshal: %w", err)
	}

	return vmConfig, nil
}

func getFields(vmStat VMStat) map[string]interface{} {
	memTotal, memUsed, memFree, memUsedPercentage := getByteMetrics(vmStat.TotalMem, vmStat.UsedMem)
	swapTotal, swapUsed, swapFree, swapUsedPercentage := getByteMetrics(vmStat.TotalSwap, vmStat.UsedSwap)
	diskTotal, diskUsed, diskFree, diskUsedPercentage := getByteMetrics(vmStat.TotalDisk, vmStat.UsedDisk)

	return map[string]interface{}{
		"status":               vmStat.Status,
		"uptime":               jsonNumberToInt64(vmStat.Uptime),
		"cpuload":              jsonNumberToFloat64(vmStat.CPULoad),
		"mem_used":             memUsed,
		"mem_total":            memTotal,
		"mem_free":             memFree,
		"mem_used_percentage":  memUsedPercentage,
		"swap_used":            swapUsed,
		"swap_total":           swapTotal,
		"swap_free":            swapFree,
		"swap_used_percentage": swapUsedPercentage,
		"disk_used":            diskUsed,
		"disk_total":           diskTotal,
		"disk_free":            diskFree,
		"disk_used_percentage": diskUsedPercentage,
	}
}

func getByteMetrics(total json.Number, used json.Number) (int64, int64, int64, float64) {
	int64Total := jsonNumberToInt64(total)
	int64Used := jsonNumberToInt64(used)
	int64Free := int64Total - int64Used
	usedPercentage := 0.0
	if int64Total != 0 {
		usedPercentage = float64(int64Used) * 100 / float64(int64Total)
	}

	return int64Total, int64Used, int64Free, usedPercentage
}

func jsonNumberToInt64(value json.Number) int64 {
	int64Value, err := value.Int64()
	if err != nil {
		return 0
	}

	return int64Value
}

func jsonNumberToFloat64(value json.Number) float64 {
	float64Value, err := value.Float64()
	if err != nil {
		return 0
	}

	return float64Value
}

func getTags(px *Proxmox, name string, vmConfig VMConfig, rt ResourceType) map[string]string {
	domain := vmConfig.Data.Searchdomain
	if len(domain) == 0 {
		domain = px.nodeSearchDomain
	}

	hostname := vmConfig.Data.Hostname
	if len(hostname) == 0 {
		hostname = name
	}
	fqdn := hostname + "." + domain

	return map[string]string{
		"node_fqdn": px.NodeName + "." + px.nodeSearchDomain,
		"vm_name":   name,
		"vm_fqdn":   fqdn,
		"vm_type":   string(rt),
	}
}
