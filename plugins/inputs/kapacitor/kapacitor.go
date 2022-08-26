package kapacitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

const (
	defaultURL = "http://localhost:9092/kapacitor/v1/debug/vars"
)

type Kapacitor struct {
	client *http.Client
	tls.ClientConfig
	URLs    []string `toml:"urls"`
	Timeout internal.Duration
}

func (*Kapacitor) Description() string {
	return "Read Kapacitor-formatted JSON metrics from one or more HTTP endpoints"
}

func (*Kapacitor) SampleConfig() string {
	return `
  instance_id = "" # unique instance identifier (REQUIRED)

  ## Multiple URLs from which to read Kapacitor-formatted JSON
  ## Default is "http://localhost:9092/kapacitor/v1/debug/vars".
  urls = [
    "http://localhost:9092/kapacitor/v1/debug/vars"
  ]

  ## Time limit for http requests
  timeout = "5s"

  ## Optional TLS Config
  # tls_ca = "/etc/circonus-unified-agent/ca.pem"
  # tls_cert = "/etc/circonus-unified-agent/cert.pem"
  # tls_key = "/etc/circonus-unified-agent/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = false
`
}

func (k *Kapacitor) Gather(ctx context.Context, acc cua.Accumulator) error {
	if k.client == nil {
		client, err := k.createHTTPClient()
		if err != nil {
			return err
		}
		k.client = client
	}

	var wg sync.WaitGroup
	for _, u := range k.URLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			if err := k.gatherURL(acc, url); err != nil {
				acc.AddError(fmt.Errorf("[url=%s]: %w", url, err))
			}
		}(u)
	}

	wg.Wait()
	return nil
}

func (k *Kapacitor) createHTTPClient() (*http.Client, error) {
	tlsCfg, err := k.ClientConfig.TLSConfig()
	if err != nil {
		return nil, fmt.Errorf("TLSConfig: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
		Timeout: k.Timeout.Duration,
	}

	return client, nil
}

type object struct {
	Values map[string]interface{} `json:"values"`
	Tags   map[string]string      `json:"tags"`
	Name   string                 `json:"name"`
}

type memstats struct {
	Alloc         int64   `json:"Alloc"`
	TotalAlloc    int64   `json:"TotalAlloc"`
	Sys           int64   `json:"Sys"`
	Lookups       int64   `json:"Lookups"`
	Mallocs       int64   `json:"Mallocs"`
	Frees         int64   `json:"Frees"`
	HeapAlloc     int64   `json:"HeapAlloc"`
	HeapSys       int64   `json:"HeapSys"`
	HeapIdle      int64   `json:"HeapIdle"`
	HeapInuse     int64   `json:"HeapInuse"`
	HeapReleased  int64   `json:"HeapReleased"`
	HeapObjects   int64   `json:"HeapObjects"`
	StackInuse    int64   `json:"StackInuse"`
	StackSys      int64   `json:"StackSys"`
	MSpanInuse    int64   `json:"MSpanInuse"`
	MSpanSys      int64   `json:"MSpanSys"`
	MCacheInuse   int64   `json:"MCacheInuse"`
	MCacheSys     int64   `json:"MCacheSys"`
	BuckHashSys   int64   `json:"BuckHashSys"`
	GCSys         int64   `json:"GCSys"`
	OtherSys      int64   `json:"OtherSys"`
	NextGC        int64   `json:"NextGC"`
	LastGC        int64   `json:"LastGC"`
	PauseTotalNs  int64   `json:"PauseTotalNs"`
	NumGC         int64   `json:"NumGC"`
	GCCPUFraction float64 `json:"GCCPUFraction"`
}

type stats struct {
	Kapacitor        *map[string]object `json:"kapacitor"`
	MemStats         *memstats          `json:"memstats"`
	ClusterID        string             `json:"cluster_id"`
	Host             string             `json:"host"`
	Product          string             `json:"product"`
	ServerID         string             `json:"server_id"`
	Version          string             `json:"version"`
	CmdLine          []string           `json:"cmdline"`
	NumEnabledTasks  int                `json:"num_enabled_tasks"`
	NumSubscriptions int                `json:"num_subscriptions"`
	NumTasks         int                `json:"num_tasks"`
}

// Gathers data from a particular URL
// Parameters:
//
//	acc    : The cua Accumulator to use
//	url    : endpoint to send request to
//
// Returns:
//
//	error: Any error that may have occurred
func (k *Kapacitor) gatherURL(
	acc cua.Accumulator,
	rurl string,
) error {
	now := time.Now()

	resp, err := k.client.Get(rurl)
	if err != nil {
		return fmt.Errorf("cli get (%s): %w", rurl, err)
	}
	defer resp.Body.Close()

	var s stats
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}

	if s.MemStats != nil {
		acc.AddFields("kapacitor_memstats",
			map[string]interface{}{
				"alloc_bytes":         s.MemStats.Alloc,
				"buck_hash_sys_bytes": s.MemStats.BuckHashSys,
				"frees":               s.MemStats.Frees,
				"gc_cpu_fraction":     s.MemStats.GCCPUFraction,
				"gc_sys_bytes":        s.MemStats.GCSys,
				"heap_alloc_bytes":    s.MemStats.HeapAlloc,
				"heap_idle_bytes":     s.MemStats.HeapIdle,
				"heap_in_use_bytes":   s.MemStats.HeapInuse,
				"heap_objects":        s.MemStats.HeapObjects,
				"heap_released_bytes": s.MemStats.HeapReleased,
				"heap_sys_bytes":      s.MemStats.HeapSys,
				"last_gc_ns":          s.MemStats.LastGC,
				"lookups":             s.MemStats.Lookups,
				"mallocs":             s.MemStats.Mallocs,
				"mcache_in_use_bytes": s.MemStats.MCacheInuse,
				"mcache_sys_bytes":    s.MemStats.MCacheSys,
				"mspan_in_use_bytes":  s.MemStats.MSpanInuse,
				"mspan_sys_bytes":     s.MemStats.MSpanSys,
				"next_gc_ns":          s.MemStats.NextGC,
				"num_gc":              s.MemStats.NumGC,
				"other_sys_bytes":     s.MemStats.OtherSys,
				"pause_total_ns":      s.MemStats.PauseTotalNs,
				"stack_in_use_bytes":  s.MemStats.StackInuse,
				"stack_sys_bytes":     s.MemStats.StackSys,
				"sys_bytes":           s.MemStats.Sys,
				"total_alloc_bytes":   s.MemStats.TotalAlloc,
			},
			map[string]string{
				"kap_version": s.Version,
				"url":         rurl,
			},
			now)
	}

	acc.AddFields("kapacitor",
		map[string]interface{}{
			"num_enabled_tasks": s.NumEnabledTasks,
			"num_subscriptions": s.NumSubscriptions,
			"num_tasks":         s.NumTasks,
		},
		map[string]string{
			"kap_version": s.Version,
			"url":         rurl,
		},
		now)

	if s.Kapacitor != nil {
		for _, obj := range *s.Kapacitor {

			// Strip out high-cardinality or duplicative tags
			excludeTags := []string{"host", "cluster_id", "server_id"}
			for _, key := range excludeTags {
				delete(obj.Tags, key)
			}

			// Convert time-related string field to int
			if _, ok := obj.Values["avg_exec_time_ns"]; ok {
				d, err := time.ParseDuration(obj.Values["avg_exec_time_ns"].(string))
				if err != nil {
					continue
				}
				obj.Values["avg_exec_time_ns"] = d.Nanoseconds()
			}

			acc.AddFields(
				"kapacitor_"+obj.Name,
				obj.Values,
				obj.Tags,
				now,
			)
		}
	}

	return nil
}

func init() {
	inputs.Add("kapacitor", func() cua.Input {
		return &Kapacitor{
			URLs:    []string{defaultURL},
			Timeout: internal.Duration{Duration: time.Second * 5},
		}
	})
}
