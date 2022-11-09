// Package circonus contains the output plugin used to output metric data to
// the Circonus platform.
package circonus

import (
	"bytes"
	"runtime/debug"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	inter "github.com/circonus-labs/circonus-unified-agent/internal"
	circmgr "github.com/circonus-labs/circonus-unified-agent/internal/circonus"
	"github.com/circonus-labs/circonus-unified-agent/plugins/outputs"
	"github.com/circonus-labs/go-trapmetrics"
)

const (
	metricVolume          = "cua_metrics_sent"
	defaultWorkerPoolSize = 2
)

var (
	agentDestination *metricDestination
	hostDestination  *metricDestination
	checkmu          sync.Mutex
)

// Circonus values are used to output data to the Circonus platform.
type Circonus struct {
	startTime time.Time
	sync.RWMutex
	Log                 cua.Logger
	DebugAPI            *bool   `toml:"debug_api"`
	TraceMetrics        *string `toml:"trace_metrics"`
	processors          processors
	DebugChecks         map[string]string `toml:"debug_checks"` // optional: use when instructed by circonus support
	metricDestinations  map[string]*metricDestination
	CacheDir            string   `toml:"cache_dir"`              // optional: where to cache the check bundle configurations - must be read/write for user running cua
	APITLSCA            string   `toml:"api_tls_ca"`             // optional: override agent.circonus api ca cert file
	APIApp              string   `toml:"api_app"`                // optional: override agent.circonus api app (default: circonus-unified-agent)
	APIURL              string   `toml:"api_url"`                // optional: override agent.circonus api url (default: https://api.circonus.com/v2)
	Broker              string   `toml:"broker"`                 // optional: override agent.circonus broker ID - numeric portion of _cid from broker api object (default is selected: enterprise or public httptrap broker)
	APIToken            string   `toml:"api_token"`              // optional: override agent.circonus api token
	CheckSearchTags     []string `toml:"check_search_tags"`      // optional: set of tags to use when searching for checks (default: service:circonus-unified-agentd)
	PoolSize            int      `toml:"pool_size"`              // size of the processor pool for a given output instance - default 2
	DebugMetrics        bool     `toml:"debug_metrics"`          // output the metrics as they are being parsed, use to verify proper parsing/tags/etc.
	SubOutput           bool     `toml:"sub_output"`             // a dedicated, special purpose, output, don't send internal cua version, etc.
	CacheConfigs        bool     `toml:"cache_configs"`          // optional: cache check bundle configurations - efficient for large number of inputs
	AllowSNMPTrapEvents bool     `toml:"allow_snmp_trap_events"` // optional: send snmp_trap text events to circonus - may result in high billing costs
}

// processors handle incoming batches
type processors struct {
	wg      sync.WaitGroup
	metrics chan []cua.Metric
}

// Init performs initialization of a Circonus client.
func (c *Circonus) Init() error {
	if !circmgr.Ready() {
		// initialize circonus metric destination manager module from config here
		cfg := &config.CirconusConfig{
			APIToken:        c.APIToken,
			APIApp:          c.APIApp,
			APIURL:          c.APIURL,
			APITLSCA:        c.APITLSCA,
			Broker:          c.Broker,
			CacheConfigs:    c.CacheConfigs,
			CacheDir:        c.CacheDir,
			DebugChecks:     c.DebugChecks,
			CheckSearchTags: c.CheckSearchTags,
		}
		if c.DebugAPI != nil {
			cfg.DebugAPI = *c.DebugAPI
		}
		if c.TraceMetrics != nil {
			cfg.TraceMetrics = *c.TraceMetrics
		}

		if err := circmgr.Initialize(cfg); err != nil {
			return err
		}
	}

	if c.PoolSize == 0 {
		c.PoolSize = defaultWorkerPoolSize
	}
	c.processors = processors{metrics: make(chan []cua.Metric)}
	c.Log.Debugf("starting %d metric processors", c.PoolSize)
	c.processors.wg.Add(c.PoolSize)
	for i := 0; i < c.PoolSize; i++ {
		i := i
		go func(id int) {
			var buf bytes.Buffer
			for m := range c.processors.metrics {
				buf.Reset()
				start := time.Now()
				nm := c.metricProcessor(id, m, buf)
				c.Log.Debugf("processor %d, processed %d metrics in %s", id, nm, time.Since(start).String())
			}
			c.processors.wg.Done()
		}(i)
	}

	return nil
}

func (p *processors) run(m []cua.Metric) {
	p.metrics <- m
}

func (p *processors) shutdown() {
	close(p.metrics)
	p.wg.Wait()
}

var sampleConfig = `
  ## Pool size - controls the number of batch processors
  ## Optional: mostly applicable to large number of inputs or inputs producing lots (100K+) of metrics
  # pool_size = 2

  ## Sub output - is this an additional output to handle specific plugin metrics (e.g. not the main, host system output)
  ## Optional - if multiple outputs think they are the main, there can be duplicate metric submissions
  # sub_output = false

  ## Debug metrics - this will output the metrics as they are being parsed - to verify parsing of names/tags/values
  ## Optional
  # debug_metrics = false
`

var description = "Configuration for Circonus output plugin."

// Conenct creates the initial check the plugin will use
func (c *Circonus) Connect() error {

	checkmu.Lock()
	defer checkmu.Unlock()

	if c.metricDestinations == nil {
		c.Lock()
		c.metricDestinations = make(map[string]*metricDestination)
		c.Unlock()
	}

	if agentDestination == nil {
		meta := circmgr.MetricMeta{
			PluginID:   "agent",
			InstanceID: config.DefaultInstanceID(),
		}
		if err := c.initMetricDestination(meta, map[string]string{}, "", ""); err != nil {
			c.Log.Errorf("unable to initialize circonus metric destination (%s)", err)
			return err
		}
		destKey := meta.Key()
		if d, ok := c.metricDestinations[destKey]; ok {
			agentDestination = d
		}
	}

	if !c.SubOutput {
		if config.DefaultPluginsEnabled() {
			if hostDestination == nil {
				meta := circmgr.MetricMeta{
					PluginID:   "host",
					InstanceID: config.DefaultInstanceID(),
				}
				if err := c.initMetricDestination(meta, map[string]string{}, "", ""); err != nil {
					c.Log.Errorf("unable to initialize circonus metric destination (%s)", err)
					return err
				}
				destKey := meta.Key()
				if d, ok := c.metricDestinations[destKey]; ok {
					hostDestination = d
				}
			}
		}
		c.emitAgentVersion()
		go func() {
			for range time.NewTicker(5 * time.Minute).C {
				c.emitAgentVersion()
				debug.FreeOSMemory()
			}
		}()
		go func() {
			for range time.NewTicker(1 * time.Minute).C {
				c.emitRuntime()
				// runtime.GC()
			}
		}()
	}

	return nil
}

func (c *Circonus) emitAgentVersion() {
	agentVersion := inter.Version()
	if agentDestination != nil {
		ts := time.Now()
		_ = agentDestination.metrics.TextSet("cua_version", nil, agentVersion, &ts)
		agentDestination.queuedMetrics++
	}
}

func (c *Circonus) emitRuntime() {
	if agentDestination != nil {
		ts := time.Now()
		_ = agentDestination.metrics.GaugeSet("cua_runtime", trapmetrics.Tags{{Category: "units", Value: "seconds"}}, time.Since(c.startTime).Seconds(), &ts)
		agentDestination.queuedMetrics++
	}
}

// Write is used to write metric data to Circonus checks.
func (c *Circonus) Write(metrics []cua.Metric) (int, error) {
	numMetrics := int64(-1)
	c.processors.run(metrics)
	return int(numMetrics), nil
}

// SampleConfig returns the sample Circonus plugin configuration.
func (c *Circonus) SampleConfig() string {
	return sampleConfig
}

// Description returns a description of the Circonus plugin configuration.
func (c *Circonus) Description() string {
	return description
}

// Close will close the Circonus client connection.
func (c *Circonus) Close() error {
	c.processors.shutdown()
	return nil
}

func init() {
	outputs.Add("circonus", func() cua.Output {
		return &Circonus{
			startTime: time.Now(),
		}
	})
}
