// Package circonus contains the output plugin used to output metric data to
// the Circonus platform.
package circonus

import (
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	inter "github.com/circonus-labs/circonus-unified-agent/internal"
	circmgr "github.com/circonus-labs/circonus-unified-agent/internal/circonus"
	"github.com/circonus-labs/circonus-unified-agent/plugins/outputs"
)

const (
	metricVolume          = "cua_metrics_sent"
	defaultWorkerPoolSize = 2
)

var (
	defaultDestination *metricDestination
	agentDestination   *metricDestination
	hostDestination    *metricDestination
	checkmu            sync.Mutex
)

// Circonus values are used to output data to the Circonus platform.
type Circonus struct {
	// for backwards compatibility, allow old config options to work
	// circonus config should be in [agent.circonus] going forward
	Broker          string            `toml:"broker"`            // optional: broker ID - numeric portion of _cid from broker api object (default is selected: enterprise or public httptrap broker)
	APIURL          string            `toml:"api_url"`           // optional: api url (default: https://api.circonus.com/v2)
	APIToken        string            `toml:"api_token"`         // api token (REQUIRED)
	APIApp          string            `toml:"api_app"`           // optional: api app (default: circonus-unified-agent)
	APITLSCA        string            `toml:"api_tls_ca"`        // optional: api ca cert file
	CacheConfigs    bool              `toml:"cache_configs"`     // optional: cache check bundle configurations - efficient for large number of inputs
	CacheDir        string            `toml:"cache_dir"`         // optional: where to cache the check bundle configurations - must be read/write for user running cua
	DebugAPI        bool              `toml:"debug_api"`         // optional: debug circonus api calls
	TraceMetrics    string            `toml:"trace_metrics"`     // optional: output json sent to broker (path to write files to or `-` for logger)
	DebugChecks     map[string]string `toml:"debug_checks"`      // optional: use when instructed by circonus support
	CheckSearchTags []string          `toml:"check_search_tags"` // optional: set of tags to use when searching for checks (default: service:circonus-unified-agentd)
	//
	// normal options, for output plugin
	//
	PoolSize           int    `toml:"pool_size"`     // size of the processor pool for a given output instance - default 2
	SubOutput          bool   `toml:"sub_output"`    // a dedicated, special purpose, output, don't send internal cua version, etc.
	DebugMetrics       bool   `toml:"debug_metrics"` // output the metrics as they are being parsed, use to verify proper parsing/tags/etc.
	OneCheck           bool   `toml:"one_check"`
	CheckNamePrefix    string `toml:"check_name_prefix"`
	metricDestinations map[string]*metricDestination
	Log                cua.Logger
	processors         processors
	sync.RWMutex
}

// processors handle incoming batches
type processors struct {
	metrics chan []cua.Metric
	wg      sync.WaitGroup
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
			DebugAPI:        c.DebugAPI,
			TraceMetrics:    c.TraceMetrics,
			DebugChecks:     c.DebugChecks,
			CheckSearchTags: c.CheckSearchTags,
		}

		if err := circmgr.Initialize(cfg, nil); err != nil {
			return err
		}
	}

	if c.CheckNamePrefix == "" {
		hn, err := os.Hostname()
		if err != nil || hn == "" {
			hn = "unknown"
		}
		c.CheckNamePrefix = hn
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
			for m := range c.processors.metrics {
				start := time.Now()
				nm := c.metricProcessor(id, m)
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
  ## One check - all metrics go to a single check vs one check per input plugin
  ## NOTE: this effectively disables automatic dashboards for supported plugins
  # one_check = false
  
  ## Pool size - controls the number of batch processors
  ## Optional: mostly applicable to large number of inputs or inputs producing lots (100K+) of metrics
  # pool_size = 2

  ## Sub output - is this an additional output to handle specific plugin metrics (e.g. not the main, host system output)
  ## Optional - if multiple outputs think they are the main, there can be duplicate metric submissions
  # sub_output = false

  ## Debug metrics - this will output the metrics as they are being parsed - to verify parsing of names/tags/values
  ## Optional
  # debug_metrics = false

  ## Check name prefix - used in check display name and check target (default: OS hostname, use with containers)
  ## Optional
  # check_name_prefix = ""
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

	if defaultDestination == nil {
		pluginID := "default"
		instanceID := ""
		metricGroupID := ""
		if err := c.initMetricDestination(pluginID, instanceID, metricGroupID); err != nil {
			c.Log.Errorf("unable to initialize circonus metric destination (%s)", err)
			return err
		}
		destKey := circmgr.MakeDestinationKey(pluginID, instanceID, metricGroupID)
		if d, ok := c.metricDestinations[destKey]; ok {
			defaultDestination = d
		}
	}

	if agentDestination == nil {
		pluginID := "agent"
		instanceID := config.DefaultInstanceID()
		metricGroupID := ""
		if err := c.initMetricDestination(pluginID, instanceID, metricGroupID); err != nil {
			c.Log.Errorf("unable to initialize circonus metric destination (%s)", err)
			return err
		}
		destKey := circmgr.MakeDestinationKey(pluginID, instanceID, metricGroupID)
		if d, ok := c.metricDestinations[destKey]; ok {
			agentDestination = d
		}
	}

	if !c.SubOutput {
		if config.DefaultPluginsEnabled() {
			if hostDestination == nil {
				pluginID := "host"
				instanceID := config.DefaultInstanceID()
				metricGroupID := ""
				if err := c.initMetricDestination(pluginID, instanceID, metricGroupID); err != nil {
					c.Log.Errorf("unable to initialize circonus metric destination (%s)", err)
					return err
				}
				destKey := circmgr.MakeDestinationKey(pluginID, instanceID, metricGroupID)
				if d, ok := c.metricDestinations[destKey]; ok {
					hostDestination = d
				}
			}
		}
		c.emitAgentVersion()
		go func() {
			for range time.NewTicker(5 * time.Minute).C {
				debug.FreeOSMemory()
				c.emitAgentVersion()
			}
		}()
	}

	return nil
}

func (c *Circonus) emitAgentVersion() {
	agentVersion := inter.Version()
	if agentDestination != nil {
		_ = agentDestination.metrics.TextSet("cua_version", nil, agentVersion, nil)
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
		return &Circonus{}
	})
}
