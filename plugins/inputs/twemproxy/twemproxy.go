package twemproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type Twemproxy struct {
	Addr  string
	Pools []string
}

var sampleConfig = `
  ## Twemproxy stats address and port (no scheme)
  addr = "localhost:22222"
  ## Monitor pool name
  pools = ["redis_pool", "mc_pool"]
`

func (t *Twemproxy) SampleConfig() string {
	return sampleConfig
}

func (t *Twemproxy) Description() string {
	return "Read Twemproxy stats data"
}

// Gather data from all Twemproxy instances
func (t *Twemproxy) Gather(ctx context.Context, acc cua.Accumulator) error {
	conn, err := net.DialTimeout("tcp", t.Addr, 1*time.Second)
	if err != nil {
		return fmt.Errorf("dial (%s): %w", t.Addr, err)
	}
	body, err := io.ReadAll(conn)
	if err != nil {
		return fmt.Errorf("readall: %w", err)
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(body, &stats); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	tags := make(map[string]string)
	tags["twemproxy"] = t.Addr
	t.processStat(acc, tags, stats)

	return nil
}

// Process Twemproxy server stats
func (t *Twemproxy) processStat(
	acc cua.Accumulator,
	tags map[string]string,
	data map[string]interface{},
) {
	if source, ok := data["source"]; ok {
		if val, ok := source.(string); ok {
			tags["source"] = val
		}
	}

	fields := make(map[string]interface{})
	metrics := []string{"total_connections", "curr_connections", "timestamp"}
	for _, m := range metrics {
		if value, ok := data[m]; ok {
			if val, ok := value.(float64); ok {
				fields[m] = val
			}
		}
	}
	acc.AddFields("twemproxy", fields, tags)

	for _, pool := range t.Pools {
		if poolStat, ok := data[pool]; ok {
			if data, ok := poolStat.(map[string]interface{}); ok {
				poolTags := copyTags(tags)
				poolTags["pool"] = pool
				t.processPool(acc, poolTags, data)
			}
		}
	}
}

// Process pool data in Twemproxy stats
func (t *Twemproxy) processPool(
	acc cua.Accumulator,
	tags map[string]string,
	data map[string]interface{},
) {
	serverTags := make(map[string]map[string]string)

	fields := make(map[string]interface{})
	for key, value := range data {
		switch key {
		case "client_connections", "forward_error", "client_err", "server_ejects", "fragments", "client_eof":
			if val, ok := value.(float64); ok {
				fields[key] = val
			}
		default:
			if data, ok := value.(map[string]interface{}); ok {
				if _, ok := serverTags[key]; !ok {
					serverTags[key] = copyTags(tags)
					serverTags[key]["server"] = key
				}
				t.processServer(acc, serverTags[key], data)
			}
		}
	}
	acc.AddFields("twemproxy_pool", fields, tags)
}

// Process backend server(redis/memcached) stats
func (t *Twemproxy) processServer(
	acc cua.Accumulator,
	tags map[string]string,
	data map[string]interface{},
) {
	fields := make(map[string]interface{})
	for key, value := range data {
		if val, ok := value.(float64); ok {
			fields[key] = val
		}
	}
	acc.AddFields("twemproxy_pool_server", fields, tags)
}

// Tags is not expected to be mutated after passing to Add.
func copyTags(tags map[string]string) map[string]string {
	newTags := make(map[string]string)
	for k, v := range tags {
		newTags[k] = v
	}
	return newTags
}

func init() {
	inputs.Add("twemproxy", func() cua.Input {
		return &Twemproxy{}
	})
}
