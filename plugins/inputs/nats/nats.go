// +build !freebsd freebsd,cgo

package nats

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	gnatsd "github.com/nats-io/nats-server/v2/server"
)

type Nats struct {
	Server          string
	ResponseTimeout internal.Duration

	client *http.Client
}

var sampleConfig = `
  ## The address of the monitoring endpoint of the NATS server
  server = "http://localhost:8222"

  ## Maximum time to receive response
  # response_timeout = "5s"
`

func (n *Nats) SampleConfig() string {
	return sampleConfig
}

func (n *Nats) Description() string {
	return "Provides metrics about the state of a NATS server"
}

func (n *Nats) Gather(acc cua.Accumulator) error {
	rurl, err := url.Parse(n.Server)
	if err != nil {
		return fmt.Errorf("url parse (%s): %w", n.Server, err)
	}
	rurl.Path = path.Join(rurl.Path, "varz")

	if n.client == nil {
		n.client = n.createHTTPClient()
	}
	resp, err := n.client.Get(rurl.String())
	if err != nil {
		return fmt.Errorf("http get (%s): %w", rurl.String(), err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("readall: %w", err)
	}

	stats := new(gnatsd.Varz)
	err = json.Unmarshal(bytes, &stats)
	if err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	acc.AddFields("nats",
		map[string]interface{}{
			"in_msgs":           stats.InMsgs,
			"out_msgs":          stats.OutMsgs,
			"in_bytes":          stats.InBytes,
			"out_bytes":         stats.OutBytes,
			"uptime":            stats.Now.Sub(stats.Start).Nanoseconds(),
			"cores":             stats.Cores,
			"cpu":               stats.CPU,
			"mem":               stats.Mem,
			"connections":       stats.Connections,
			"total_connections": stats.TotalConnections,
			"subscriptions":     stats.Subscriptions,
			"slow_consumers":    stats.SlowConsumers,
			"routes":            stats.Routes,
			"remotes":           stats.Remotes,
		},
		map[string]string{"server": n.Server},
		time.Now())

	return nil
}

func (n *Nats) createHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	timeout := n.ResponseTimeout.Duration
	if timeout == time.Duration(0) {
		timeout = 5 * time.Second
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

func init() {
	inputs.Add("nats", func() cua.Input {
		return &Nats{
			Server: "http://localhost:8222",
		}
	})
}
