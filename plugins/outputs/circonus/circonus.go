// Package circonus contains the output plugin used to output metric data to
// the Circonus platform.
package circonus

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	cgm "github.com/circonus-labs/circonus-gometrics/v3"
	"github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	inter "github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/outputs"
	apiclient "github.com/circonus-labs/go-apiclient"
	apiclicfg "github.com/circonus-labs/go-apiclient/config"
)

const (
	metricVolume = "cua_metrics_sent"
)

var (
	defaultCheck *cgm.CirconusMetrics
	agentCheck   *cgm.CirconusMetrics
	hostCheck    *cgm.CirconusMetrics
	brokerTLS    *tls.Config
	checkmu      sync.Mutex
)

// Circonus values are used to output data to the Circonus platform.
type Circonus struct {
	Broker          string `toml:"broker"`
	APIURL          string `toml:"api_url"`
	APIToken        string `toml:"api_token"`
	APIApp          string `toml:"api_app"`
	APITLSCA        string `toml:"api_tls_ca"`
	OneCheck        bool   `toml:"one_check"`
	CheckNamePrefix string `toml:"check_name_prefix"`
	CacheConfigs    bool   `toml:"cache_configs"` // cache check bundle configurations - efficient for large number of inputs
	CacheDir        string `toml:"cache_dir"`     // where to cache the check bundle configurations - must be read/write for user running cua
	PoolSize        int    `toml:"pool_size"`     // size of the processor pool for a given output instance - default 2
	// hidden troubleshooting/tuning parameters
	DebugCGM        bool   `toml:"debug_cgm"`        // debug cgm interactions with api and broker
	DumpCGMMetrics  bool   `toml:"dump_cgm_metrics"` // dump the actual JSON being sent to the broker by cgm
	DebugMetrics    bool   `toml:"debug_metrics"`    // output the metrics as they are being sent to cgm, use to verify proper parsing/tags/etc.
	SubOutput       bool   `toml:"sub_output"`       // a dedicated, special purpose, output, don't send internal cua version, etc.
	DynamicSubmit   bool   `toml:"dynamic_submit"`   // control cgm auto-submissions, or manually at end of batch processing
	DynamicInterval string `toml:"dynamic_interval"` // on what interval should the dynamic cgm instances submit, default 10s
	apicfg          apiclient.Config
	checks          map[string]*cgm.CirconusMetrics
	brokerTLS       *tls.Config
	Log             cua.Logger
	processors      processors
	sync.RWMutex
}

// processors handle incoming batches
type processors struct {
	metrics chan []cua.Metric
	wg      sync.WaitGroup
}

// Init performs initialization of a Circonus client.
func (c *Circonus) Init() error {

	if c.CacheConfigs && c.CacheDir == "" {
		c.Log.Warn("cache_configs on, cache_dir not set, disabling configuration caching")
		c.CacheConfigs = false
	}
	if c.CacheConfigs && c.CacheDir != "" {
		info, err := os.Stat(c.CacheDir)
		if err != nil {
			c.Log.Warnf("cache_dir (%s): %s, disabling configuration caching", c.CacheDir, err)
			c.CacheConfigs = false
			if !info.IsDir() {
				c.Log.Warnf("cache_dir (%s): not a directory, disabling configuration caching", c.CacheDir, err)
				c.CacheConfigs = false
			}
		}
	}

	if c.APIToken == "" {
		return fmt.Errorf("circonus api token is required")
	}

	if c.APIApp == "" {
		c.APIApp = "circonus-unified-agent"
	}

	c.apicfg = apiclient.Config{
		TokenKey: c.APIToken,
		TokenApp: c.APIApp,
	}

	if c.APIURL != "" {
		c.apicfg.URL = c.APIURL
	}

	if c.APITLSCA != "" {
		cp := x509.NewCertPool()
		cert, err := os.ReadFile(c.APITLSCA)
		if err != nil {
			return fmt.Errorf("unable to load api ca file (%s): %w", c.APITLSCA, err)
		}
		if !cp.AppendCertsFromPEM(cert) {
			return fmt.Errorf("unable to parse api ca file (%s): %w", c.APITLSCA, err)
		}
		c.apicfg.CACert = cp
	}

	if c.Broker != "" {
		c.Broker = strings.Replace(c.Broker, "/broker/", "", 1)
	}

	if c.CheckNamePrefix == "" {
		hn, err := os.Hostname()
		if err != nil || hn == "" {
			hn = "unknown"
		}
		c.CheckNamePrefix = hn
	}

	if c.PoolSize == 0 {
		c.PoolSize = 2 // runtime.NumCPU()
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
  ## Circonus API token must be provided to use this plugin:
  api_token = ""

  ## Circonus API application (associated with token):
  ## example:
  # api_app = "circonus-unified-agent"

  ## Circonus API URL:
  ## example:
  # api_url = "https://api.circonus.com/"

  ## Circonus API TLS CA file, optional, for internal deployments with private certificates: 
  ## example:
  # api_tls_ca = "/opt/circonus/unified-agent/etc/circonus_api_ca.pem"

  ## Check name prefix - unique prefix to use for all checks created by this instance
  ## default is the hostname from the OS. If set, "host" tag on metrics will be 
  ## overridden with this value. For containers, use omit_hostname=true in agent section
  ## and set this value, so that the plugin will be able to predictively find the check 
  ## for this instance. Otherwise, the container's os.Hostname() will be used
  ## (resulting in a new check being created every time the container starts).
  ## example:
  # check_name_prefix = "example"

  ## One check - all metrics go to a single check vs one check per input plugin
  ## NOTE: this effectively disables automatic dashboards for supported plugins
  # one_check = false
  
  ## Broker
  ## Optional: explicit broker id or blank (default blank, auto select)
  ## example:
  # broker = "/broker/35"

  ## Performance optimization with lots of plugins (or instances of plugins)
  ## Optional: cache the check configurations
  ## example:
  # cache_configs = true
  ## Note: cache_dir must be read/write for the user running the cua process
  # cache_dir = "/opt/circonus/etc/cache.d"

  ## Pool size controls the number of batch processors
  ## Optional: mostly applicable to large number of inputs or inputs producing lots (100K+) of metrics
  # pool_size = 2
`

var description = "Configuration for Circonus output plugin."

// Conenct creates the initial check the plugin will use
func (c *Circonus) Connect() error {
	if c.APIToken == "" {
		c.Log.Error("Circonus API Token is required, unable to initialize check(s)")
		return nil
	}

	checkmu.Lock()
	defer checkmu.Unlock()

	if c.checks == nil {
		c.Lock()
		c.checks = make(map[string]*cgm.CirconusMetrics)
		c.Unlock()
	}

	if defaultCheck == nil {
		if err := c.initCheck("*", "", ""); err != nil {
			c.Log.Errorf("unable to initialize circonus check (%s)", err)
			return err
		}

		if d, ok := c.checks["*"]; ok {
			defaultCheck = d
			if brokerTLS == nil {
				brokerTLS = d.GetBrokerTLSConfig()
			}
		}
	}

	if brokerTLS != nil {
		c.brokerTLS = brokerTLS
	}

	if agentCheck == nil {
		if err := c.initCheck("agent", "agent", ""); err != nil {
			c.Log.Errorf("unable to initialize circonus check (%s)", err)
			return err
		}

		if d, ok := c.checks["agent"]; ok {
			agentCheck = d
		}
	}

	if !c.SubOutput {
		if config.DefaultPluginsEnabled() {
			if hostCheck == nil {
				if err := c.initCheck("host", "host", ""); err != nil {
					c.Log.Errorf("unable to initialize circonus check (%s)", err)
					return err
				}

				if d, ok := c.checks["host"]; ok {
					hostCheck = d
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
	if agentCheck != nil {
		agentVersion := inter.Version()
		agentCheck.SetText("cua_version", agentVersion)
	}
}

func (c *Circonus) metricProcessor(id int, metrics []cua.Metric) int64 {
	c.Log.Debugf("processor %d, received %d batches", id, len(metrics))
	start := time.Now()
	numMetrics := int64(0)
	for _, m := range metrics {
		switch m.Type() {
		case cua.Counter, cua.Gauge, cua.Summary:
			numMetrics += c.buildNumerics(m)
		case cua.Untyped:
			fields := m.FieldList()
			if s, ok := fields[0].Value.(string); ok {
				if strings.Contains(s, "H[") && strings.Contains(s, "]=") {
					numMetrics += c.buildHistogram(m)
				} else {
					numMetrics += c.buildTexts(m)
				}
			} else {
				numMetrics += c.buildNumerics(m)
			}
		case cua.Histogram:
			numMetrics += c.buildHistogram(m)
		case cua.CumulativeHistogram:
			numMetrics += c.buildCumulativeHistogram(m)
		default:
			c.Log.Warnf("processor %d, unknown type %T, ignoring", id, m)
		}
	}
	if agentCheck != nil {
		agentCheck.RecordValue(metricVolume, float64(numMetrics))
		numMetrics++
		if !c.SubOutput {
			agentCheck.AddGauge(metricVolume+"_batch", numMetrics)
			numMetrics++
		}
	}
	c.Log.Debugf("processor %d, queued %d metrics for submission in %s", id, numMetrics, time.Since(start).String())

	if !c.DynamicSubmit {
		sendStart := time.Now()
		var wg sync.WaitGroup
		c.RLock()
		wg.Add(len(c.checks))
		for _, dest := range c.checks {
			go func(d *cgm.CirconusMetrics) {
				defer wg.Done()
				d.Flush()
			}(dest)
		}
		wg.Wait()
		c.RUnlock()
		c.Log.Debugf("processor %d, non-dynamic submit: sent metrics in %s", id, time.Since(sendStart))
	}

	return numMetrics
}

// Write is used to write metric data to Circonus checks.
func (c *Circonus) Write(metrics []cua.Metric) (int, error) {
	if c.APIToken == "" {
		return 0, fmt.Errorf("Circonus API Token is required, dropping metrics")
	}

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

//
// circonus specific methods
//

// getMetricDest returns cgm instance for the plugin identified by a plugin and plugin instance id
func (c *Circonus) getMetricDest(m cua.Metric) *cgm.CirconusMetrics {
	plugin := m.Origin()
	instanceID := m.OriginInstance()

	// default - used in two cases:
	// 1. plugin cannot be identified
	// 2. user as enabled one_check
	var defaultDest *cgm.CirconusMetrics
	if defaultCheck != nil {
		defaultDest = defaultCheck
	}

	if c.OneCheck || plugin == "" {
		return defaultDest
	}

	if config.IsDefaultInstanceID(instanceID) {
		// host metrics - the "default" plugins which are enabled by default
		// but can be controlled via the (ENABLE_DEFAULT_PLUGINS env var
		// any value other than "false" will enable the default plugins)
		if config.IsDefaultPlugin(plugin) {
			hostDest := defaultDest
			if config.DefaultPluginsEnabled() {
				if hostCheck != nil {
					hostDest = hostCheck
				}
			}
			return hostDest
		}
		// agent metrics - metrics the agent emits about itself - always enabled
		if config.IsAgentPlugin(plugin) {
			agentDest := defaultDest
			if agentCheck != nil {
				agentDest = agentCheck
			}
			return agentDest
		}
	}

	id := plugin
	if instanceID != "" {
		id += ":" + instanceID
	}

	// otherwise - find (or create) a check for the specific plugin

	c.RLock()
	d, found := c.checks[id]
	c.RUnlock()

	if found {
		return d
	}

	if err := c.initCheck(id, plugin+" "+instanceID, instanceID); err == nil {
		if d, ok := c.checks[id]; ok {
			return d
		}
	} else {
		c.Log.Errorf("error initializing check: %s", err)
		os.Exit(1) //nolint:gocritic
	}

	return defaultDest
}

// logshim is for cgm - it uses the info level cgm and
// agent debug logging are controlled independently
type logshim struct {
	logh   cua.Logger
	prefix string
}

func (l logshim) Printf(fmt string, args ...interface{}) {
	if len(args) == 0 {
		l.logh.Info(l.prefix + ": " + fmt)
	} else {
		l.logh.Infof(l.prefix+": "+fmt, args...)
	}
}

// initCheck initializes cgm instance for the plugin identified by id
func (c *Circonus) initCheck(id, name, instanceID string) error {
	c.Lock()
	defer c.Unlock()

	plugID := id
	if id == "*" {
		plugID = "default"
		name = "default"
	}

	checkConfigFile := ""
	submissionURL := ""
	saveConfig := false

	if c.CacheConfigs && instanceID != "" {
		path := c.CacheDir
		if path != "" {
			checkConfigFile = filepath.Join(path, instanceID+".json")
			data, err := os.ReadFile(checkConfigFile)
			if err != nil {
				if !os.IsNotExist(err) {
					c.Log.Warnf("unable to read %s: %s", checkConfigFile, err)
					checkConfigFile = ""
				}
			} else {
				var b apiclient.CheckBundle
				if err := json.Unmarshal(data, &b); err != nil {
					c.Log.Warnf("parsing check config %s: %s", checkConfigFile, err)
					checkConfigFile = ""
				}
				submissionURL = b.Config[apiclicfg.SubmissionURL]
				c.Log.Debugf("using cached config: %s - %s", checkConfigFile, submissionURL)
			}
		}
	}

	checkType := "httptrap:cua:" + plugID + ":" + runtime.GOOS

	cfg := &cgm.Config{}
	cfg.Debug = c.DebugCGM
	cfg.DumpMetrics = c.DumpCGMMetrics
	if c.DebugCGM || c.DumpCGMMetrics {
		cfg.Log = logshim{
			logh:   c.Log,
			prefix: plugID,
		}
	}
	if !c.DynamicSubmit {
		cfg.Interval = "0" // submit on completion of batch to Write
	} else if c.DynamicInterval != "" {
		c.Log.Debugf("setting dynamic submit interval to %q", c.DynamicInterval)
		cfg.Interval = c.DynamicInterval
	}
	cfg.CheckManager.SerialInit = true
	if brokerTLS != nil {
		cfg.CheckManager.Broker.TLSConfig = brokerTLS
	}
	if submissionURL != "" {
		cfg.CheckManager.Check.SubmissionURL = submissionURL
	} else {
		saveConfig = true
		cfg.CheckManager.API = c.apicfg
		if c.Broker != "" {
			cfg.CheckManager.Broker.ID = c.Broker
		}
		cfg.CheckManager.Check.InstanceID = strings.Replace(checkType, "httptrap", c.CheckNamePrefix, 1)
		cfg.CheckManager.Check.TargetHost = c.CheckNamePrefix
		cfg.CheckManager.Check.DisplayName = c.CheckNamePrefix + " " + name + " (" + runtime.GOOS + ")"
		cfg.CheckManager.Check.Type = checkType
		_, an := filepath.Split(os.Args[0])
		cfg.CheckManager.Check.SearchTag = "service:" + an
	}

	m, err := cgm.New(cfg)
	if err != nil {
		return fmt.Errorf("initializing cgm instance for %s (%w)", id, err)
	}

	if !m.Ready() {
		ticker := time.NewTicker(250 * time.Millisecond)
		for range ticker.C {
			if m.Ready() {
				ticker.Stop()
				break
			}
		}
	}

	if c.CacheConfigs && saveConfig {
		bundle := m.GetCheckBundle()
		if checkConfigFile != "" && bundle != nil {
			data, err := json.Marshal(bundle)
			if err != nil {
				c.Log.Warnf("marshal check conf: %s", err)
			} else if err := os.WriteFile(checkConfigFile, data, 0644); err != nil { //nolint:gosec
				c.Log.Warnf("save check conf %s: %s", checkConfigFile, err)
			}
		}
	}

	c.checks[id] = m
	return nil
}

// buildNumerics constructs numeric metrics from a cua metric.
func (c *Circonus) buildNumerics(m cua.Metric) int64 {
	dest := c.getMetricDest(m)
	if dest == nil {
		c.Log.Warnf("no check destination found for metric (%#v)", m)
		return 0
	}
	numMetrics := int64(0)
	tags := c.convertTags(m)
	for _, field := range m.FieldList() {
		mn := strings.TrimSuffix(field.Key, "__value")
		if c.DebugMetrics {
			c.Log.Infof("%s %v %v %T\n", mn, tags, field.Value, field.Value)
		}
		switch v := field.Value.(type) {
		case string:
			dest.SetTextWithTags(mn, tags, v)
		default:
			// don't aggregate - throws of stuff
			// dest.AddGaugeWithTags(mn, tags, v)
			dest.SetGaugeWithTags(mn, tags, v)
		}
		numMetrics++
	}

	return numMetrics
}

// buildTexts constructs text metrics from a cua metric.
func (c *Circonus) buildTexts(m cua.Metric) int64 {
	dest := c.getMetricDest(m)
	if dest == nil {
		c.Log.Warnf("no check destination found for metric (%#v)", m)
		return 0
	}
	numMetrics := int64(0)
	tags := c.convertTags(m)

	for _, field := range m.FieldList() {
		mn := strings.TrimSuffix(field.Key, "__value")
		if c.DebugMetrics {
			c.Log.Infof("%s %v %v %T\n", mn, tags, field.Value, field.Value)
		}
		switch v := field.Value.(type) {
		case string:
			dest.SetTextWithTags(mn, tags, v)
		default:
			// don't aggregate - throws of stuff
			// dest.AddGaugeWithTags(mn, tags, v)
			dest.SetGaugeWithTags(mn, tags, v)
		}
		numMetrics++
	}

	return numMetrics
}

// buildHistogram constructs histogram metrics from a cua metric.
func (c *Circonus) buildHistogram(m cua.Metric) int64 {
	dest := c.getMetricDest(m)
	if dest == nil {
		c.Log.Warnf("no check destination found for metric (%#v)", m)
		return 0
	}

	numMetrics := int64(0)
	mn := strings.TrimSuffix(m.Name(), "__value")
	tags := c.convertTags(m)

	for _, field := range m.FieldList() {
		v, err := strconv.ParseFloat(field.Key, 64)
		if err != nil {
			c.Log.Errorf("cannot parse histogram (%s) field.key (%s) as float: %s\n", mn, field.Key, err)
			continue
		}
		if c.DebugMetrics {
			c.Log.Infof("%s %v v:%v vt%T n:%v nT:%T\n", mn, tags, v, v, field.Value, field.Value)
		}

		dest.RecordCountForValueWithTags(mn, tags, v, field.Value.(int64))
		numMetrics++
	}

	return numMetrics
}

// buildCumulativeHistogram constructs cumulative histogram metrics from a cua metric.
func (c *Circonus) buildCumulativeHistogram(m cua.Metric) int64 {
	dest := c.getMetricDest(m)
	if dest == nil {
		c.Log.Warnf("no check destination found for metric (%#v)", m)
		return 0
	}

	numMetrics := int64(0)
	mn := strings.TrimSuffix(m.Name(), "__value")
	tags := c.convertTags(m)

	buckets := make([]string, 0)

	for _, field := range m.FieldList() {
		v, err := strconv.ParseFloat(field.Key, 64)
		if err != nil {
			c.Log.Errorf("cannot parse histogram (%s) field.key (%s) as float: %s\n", mn, field.Key, err)
			continue
		}
		if c.DebugMetrics {
			c.Log.Infof("%s %v v:%v vt%T n:%v nT:%T\n", mn, tags, v, v, field.Value, field.Value)
		}

		buckets = append(buckets, fmt.Sprintf("H[%e]=%d", v, field.Value))
		numMetrics++
	}

	_ = dest.Custom(dest.MetricNameWithStreamTags(mn, tags), cgm.Metric{
		Type:  cgm.MetricTypeCumulativeHistogram,
		Value: buckets, // buckets are submitted as a string array
	})
	if c.DebugMetrics {
		c.Log.Infof("%s|ST[%s] %s\n", mn, tags, strings.Join(buckets, "\n"))
	}

	return numMetrics
}

// convertTags reformats cua tags to cgm tags
func (c *Circonus) convertTags(m cua.Metric) cgm.Tags { //nolint:unparam
	var ctags cgm.Tags

	tags := m.TagList()

	if len(tags) == 0 && m.Origin() == "" {
		return ctags
	}

	ctags = make(cgm.Tags, 0)

	if len(tags) > 0 {
		for _, t := range tags {
			// if t.Key == "alias" {
			// 	continue
			// }
			ctags = append(ctags, cgm.Tag{Category: t.Key, Value: t.Value})
		}
	}

	if m.Origin() != "" {
		// from config file `inputs.*`, the part after period
		ctags = append(ctags, cgm.Tag{Category: "input_plugin", Value: m.Origin()})
	}
	if m.Name() != "" && m.Name() != m.Origin() {
		// what the plugin identifies a subgroup of metrics as, some have multiple names
		// e.g. internal, smart, aws, etc.
		ctags = append(ctags, cgm.Tag{Category: "input_metric_group", Value: m.Name()})
	}

	// this is included in the check type/display name now so it doesn't need to be a tag
	// if m.OriginInstance() != "" {
	// 	ctags = append(ctags, cgm.Tag{Category: "input_instance_id", Value: m.OriginInstance()})
	// }

	return ctags
}
