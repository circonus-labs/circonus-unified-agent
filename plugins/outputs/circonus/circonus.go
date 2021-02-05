// Package circonus contains the output plugin used to output metric data to
// the Circonus platform.
package circonus

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	cgm "github.com/circonus-labs/circonus-gometrics/v3"
	"github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	inter "github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/outputs"
	apiclient "github.com/circonus-labs/go-apiclient"
)

const (
	metricVolume = "cua_metrics_sent"
)

// Circonus values are used to output data to the Circonus platform.
type Circonus struct {
	Broker           string `toml:"broker"`
	APIURL           string `toml:"api_url"`
	APIToken         string `toml:"api_token"`
	APIApp           string `toml:"api_app"`
	APITLSCA         string `toml:"api_tls_ca"`
	OneCheck         bool   `toml:"one_check"`
	CGMFlushInterval string `toml:"cgm_flush_interval"`
	CheckNamePrefix  string `toml:"check_name_prefix"`
	DebugCGM         bool   `toml:"debug_cgm"`
	DebugMetrics     bool   `toml:"debug_metrics"`
	apicfg           apiclient.Config
	checks           map[string]*cgm.CirconusMetrics
	Log              cua.Logger
}

// Init performs initialization of a Circonus client.
func (c *Circonus) Init() error {

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
		cert, err := ioutil.ReadFile(c.APITLSCA)
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

	if c.CGMFlushInterval != "" {
		interval, err := time.ParseDuration(c.CGMFlushInterval)
		if err != nil {
			return fmt.Errorf("invalid cgm flush interval (%s): %w", c.CGMFlushInterval, err)
		}
		if interval == time.Duration(0) {
			return fmt.Errorf("invalid cgm flush interval (%s), must be >0", c.CGMFlushInterval)
		}
	}

	return nil
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
  # one_check = false
  
  ## Broker
  ## Optional: explicit broker id or blank (default blank, auto select)
  ## example:
  # broker = "/broker/35"
`

var description = "Configuration for Circonus output plugin."

// Conenct creates the initial check the plugin will use
func (c *Circonus) Connect() error {
	if c.APIToken == "" {
		c.Log.Error("Circonus API Token is required, unable to initialize check(s)")
		return nil
	}

	if c.checks == nil {
		c.checks = make(map[string]*cgm.CirconusMetrics)
	}

	if err := c.initCheck("*", ""); err != nil {
		c.Log.Errorf("unable to initialize circonus check (%s)", err)
		return err
	}

	c.emitAgentVersion()
	go func() {
		for range time.NewTicker(5 * time.Minute).C {
			c.emitAgentVersion()
		}
	}()

	return nil
}

func (c *Circonus) emitAgentVersion() {
	if defaultDest := c.checks["*"]; defaultDest != nil {
		agentVersion := inter.Version()
		defaultDest.SetText("cua_version", agentVersion)
	}
}

// Write is used to write metric data to Circonus checks.
func (c *Circonus) Write(metrics []cua.Metric) error {
	if c.APIToken == "" {
		return fmt.Errorf("Circonus API Token is required, dropping metrics")
	}

	defaultDest := c.checks["*"]
	numMetrics := int64(0)
	for _, m := range metrics {
		switch m.Type() {
		case cua.Counter, cua.Gauge, cua.Summary:
			numMetrics += c.buildNumerics(defaultDest, m)
		case cua.Untyped:
			fields := m.FieldList()
			if s, ok := fields[0].Value.(string); ok {
				if strings.Contains(s, "H[") && strings.Contains(s, "]=") {
					numMetrics += c.buildHistogram(defaultDest, m)
				} else {
					numMetrics += c.buildTexts(defaultDest, m)
				}
			} else {
				numMetrics += c.buildNumerics(defaultDest, m)
			}
		case cua.Histogram:
			numMetrics += c.buildHistogram(defaultDest, m)
		case cua.CumulativeHistogram:
			numMetrics += c.buildCumulativeHistogram(defaultDest, m)
		default:
		}
	}
	defaultDest.AddGauge(metricVolume+"_batch", numMetrics)
	defaultDest.RecordValue(metricVolume, float64(numMetrics))
	c.Log.Debugf("queued %d metrics for submission", numMetrics)

	return nil
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
func (c *Circonus) getMetricDest(defaultDest *cgm.CirconusMetrics, plugin, instanceID string) *cgm.CirconusMetrics {
	if c.OneCheck || plugin == "" {
		return defaultDest
	}
	if config.IsDefaultPlugin(plugin) && config.IsDefaultInstanceID(instanceID) {
		return defaultDest
	}

	id := plugin
	if instanceID != "" {
		id += ":" + instanceID
	}

	if d, ok := c.checks[id]; ok {
		return d
	}

	if err := c.initCheck(id, plugin+" "+instanceID); err == nil {
		if d, ok := c.checks[id]; ok {
			return d
		}
	}

	return defaultDest
}

type logshim struct {
	logh cua.Logger
}

func (l logshim) Printf(fmt string, args ...interface{}) {
	if len(args) == 0 {
		l.logh.Info(fmt)
	} else {
		l.logh.Infof(fmt, args...)
	}
}

// initCheck initializes cgm instance for the plugin identified by id
func (c *Circonus) initCheck(id, name string) error {
	plugID := id
	if id == "*" {
		plugID = "default"
		name = "default"
	}

	checkType := "httptrap:cua:" + plugID + ":" + runtime.GOOS

	cfg := &cgm.Config{}
	cfg.Debug = c.DebugCGM
	if c.DebugCGM {
		cfg.Log = logshim{logh: c.Log}
	}
	if c.CGMFlushInterval != "" {
		cfg.Interval = c.CGMFlushInterval
	}
	cfg.CheckManager.API = c.apicfg
	if c.Broker != "" {
		cfg.CheckManager.Broker.ID = c.Broker
	}
	cfg.CheckManager.Check.InstanceID = strings.Replace(checkType, "httptrap", c.CheckNamePrefix, 1)
	cfg.CheckManager.Check.TargetHost = c.CheckNamePrefix
	cfg.CheckManager.Check.DisplayName = c.CheckNamePrefix + " " + name + " (" + runtime.GOOS + ")"
	cfg.CheckManager.Check.Type = checkType

	m, err := cgm.New(cfg)
	if err != nil {
		return fmt.Errorf("initializing cgm instance for %s (%w)", id, err)
	}

	c.checks[id] = m
	return nil
}

// buildNumerics constructs numeric metrics from a cua metric.
func (c *Circonus) buildNumerics(defaultDest *cgm.CirconusMetrics, m cua.Metric) int64 {
	dest := c.getMetricDest(defaultDest, m.Name(), m.OriginInstance())
	if dest == nil {
		// no default and no plugin specific
		return 0
	}
	numMetrics := int64(0)
	tags := c.convertTags(m.Origin(), m.OriginInstance(), m.TagList())
	for _, field := range m.FieldList() {
		mn := strings.TrimSuffix(field.Key, "__value")
		if c.DebugMetrics {
			c.Log.Infof("%s %v %v %T\n", mn, tags, field.Value, field.Value)
		}
		dest.AddGaugeWithTags(mn, tags, field.Value)
		numMetrics++
	}

	return numMetrics
}

// buildTexts constructs text metrics from a cua metric.
func (c *Circonus) buildTexts(defaultDest *cgm.CirconusMetrics, m cua.Metric) int64 {
	dest := c.getMetricDest(defaultDest, m.Name(), m.OriginInstance())
	if dest == nil {
		// no default and no plugin specific
		return 0
	}
	numMetrics := int64(0)
	tags := c.convertTags(m.Origin(), m.OriginInstance(), m.TagList())
	for _, field := range m.FieldList() {
		mn := strings.TrimSuffix(field.Key, "__value")
		if c.DebugMetrics {
			c.Log.Infof("%s %v %v %T\n", mn, tags, field.Value, field.Value)
		}
		switch v := field.Value.(type) {
		case string:
			dest.SetTextWithTags(mn, tags, v)
		default:
			dest.AddGaugeWithTags(mn, tags, v)
		}
		numMetrics++
	}

	return numMetrics
}

// buildHistogram constructs histogram metrics from a cua metric.
func (c *Circonus) buildHistogram(defaultDest *cgm.CirconusMetrics, m cua.Metric) int64 {
	dest := c.getMetricDest(defaultDest, m.Name(), m.OriginInstance())
	if dest == nil {
		// no default and no plugin specific
		return 0
	}

	numMetrics := int64(0)
	mn := strings.TrimSuffix(m.Name(), "__value")
	tags := c.convertTags(m.Origin(), m.OriginInstance(), m.TagList())

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
func (c *Circonus) buildCumulativeHistogram(defaultDest *cgm.CirconusMetrics, m cua.Metric) int64 {
	dest := c.getMetricDest(defaultDest, m.Name(), m.OriginInstance())
	if dest == nil {
		// no default and no plugin specific
		return 0
	}

	numMetrics := int64(0)
	mn := strings.TrimSuffix(m.Name(), "__value")
	tags := c.convertTags(m.Origin(), m.OriginInstance(), m.TagList())

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
func (c *Circonus) convertTags(pluginName, pluginInstanceID string, tags []*cua.Tag) cgm.Tags { //nolint:unparam
	var ctags cgm.Tags

	if len(tags) == 0 && pluginName == "" {
		return ctags
	}

	ctags = make(cgm.Tags, 0)

	if len(tags) > 0 {
		for _, t := range tags {
			if t.Key == "alias" {
				continue
			}
			if t.Key == "host" && c.CheckNamePrefix != "" {
				ctags = append(ctags, cgm.Tag{Category: t.Key, Value: c.CheckNamePrefix})
				continue
			}
			ctags = append(ctags, cgm.Tag{Category: t.Key, Value: t.Value})
		}
	}

	if pluginName != "" {
		ctags = append(ctags, cgm.Tag{Category: "input_plugin", Value: pluginName})
	}

	// if pluginInstanceID != "" {
	// 	ctags = append(ctags, cgm.Tag{Category: "input_instance_id", Value: pluginInstanceID})
	// }

	return ctags
}
