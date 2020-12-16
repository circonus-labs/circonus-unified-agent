// Package circonus contains the output plugin used to output metric data to
// the Circonus platform.
package circonus

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	cgm "github.com/circonus-labs/circonus-gometrics/v3"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/outputs"
	apiclient "github.com/circonus-labs/go-apiclient"
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
	DebugCGM        bool   `toml:"debug_cgm"`
	apicfg          apiclient.Config
	checks          map[string]*cgm.CirconusMetrics
	Log             cua.Logger
}

// Init performs initialization of a Circonus client.
func (c *Circonus) Init() error {

	if c.APIToken == "" {
		c.Log.Error("Circonus API Token is required")
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

	if c.Broker != "" {
		c.Broker = strings.Replace(c.Broker, "/broker/", "", 1)
	}

	if c.CheckNamePrefix == "" {
		hn, err := os.Hostname()
		if err != nil {
			hn = "unknown"
		}
		c.CheckNamePrefix = hn
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
  # api_tls_ca = "/etc/circonus-unified-agent/circonus_api_ca.pem"

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

// Conenct creates the client connection to the Circonus broker.
func (c *Circonus) Connect() error {
	if c.APIToken == "" {
		c.Log.Error("Circonus API Token is required, unable to initialize check(s)")
		return nil
	}

	if c.checks == nil {
		c.checks = make(map[string]*cgm.CirconusMetrics)
	}

	if err := c.initCheck("*"); err != nil {
		c.Log.Errorf("unable to initialize circonus check (%s)", err)
		return err
	}

	// to get agent metrics, turn on 'internal' input
	return nil
}

// Write is used to write metric data to Circonus checks.
func (c *Circonus) Write(metrics []cua.Metric) error {
	if c.APIToken == "" {
		return fmt.Errorf("Circonus API Token is required, dropping metrics")
		// c.Log.Error("Circonus API Token is required")
		// return nil
	}

	defaultDest := c.checks["*"]
	for _, m := range metrics {
		switch m.Type() {
		case cua.Counter, cua.Gauge, cua.Summary:
			c.buildNumerics(defaultDest, m)
		case cua.Untyped:
			fields := m.FieldList()
			if s, ok := fields[0].Value.(string); ok {
				if strings.Contains(s, "H[") && strings.Contains(s, "]=") {
					c.buildHistogram(defaultDest, m)
				} else {
					c.buildTexts(defaultDest, m)
				}
			} else {
				c.buildNumerics(defaultDest, m)
			}
		case cua.Histogram:
			c.buildHistogram(defaultDest, m)
		default:
		}
	}
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

// getMetricDest returns cgm instance for the plugin identified by id
func (c *Circonus) getMetricDest(defaultDest *cgm.CirconusMetrics, id string) *cgm.CirconusMetrics {
	if c.OneCheck || id == "" {
		return defaultDest
	}

	if d, ok := c.checks[id]; ok {
		return d
	}

	if err := c.initCheck(id); err == nil {
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
		l.logh.Debug(fmt)
	} else {
		l.logh.Debugf(fmt, args)
	}
}

// initCheck initializes cgm instance for the plugin identified by id
func (c *Circonus) initCheck(id string) error {
	checkType := "httptrap:cua:"

	if id == "*" {
		checkType += "default"
	} else {
		checkType += id
	}

	cfg := &cgm.Config{}
	cfg.Debug = c.DebugCGM
	if c.DebugCGM {
		cfg.Log = logshim{logh: c.Log}
	}
	cfg.CheckManager.API = c.apicfg
	if c.Broker != "" {
		cfg.CheckManager.Broker.ID = c.Broker
	}
	cfg.CheckManager.Check.InstanceID = strings.Replace(checkType, "httptrap", c.CheckNamePrefix, 1)
	cfg.CheckManager.Check.Type = checkType

	m, err := cgm.New(cfg)
	if err != nil {
		return fmt.Errorf("initializing cgm instance for %s (%w)", id, err)
	}

	c.checks[id] = m
	return nil
}

// buildNumerics constructs numeric metrics from a cua metric.
func (c *Circonus) buildNumerics(defaultDest *cgm.CirconusMetrics, m cua.Metric) {
	fields := m.FieldList()
	for _, field := range fields {
		mn := m.Name() + "." + field.Key
		inputPluginName := ""
		mnp := strings.SplitN(mn, ".", 2)
		if len(mnp) == 2 {
			inputPluginName = mnp[0]
			mn = mnp[1]
		}

		dest := c.getMetricDest(defaultDest, inputPluginName)
		if dest == nil {
			// no default and no plugin specific
			return
		}

		if strings.HasSuffix(mn, "__value") {
			mn = mn[:len(mn)-7]
		}

		dest.AddGaugeWithTags(mn, c.convertTags(inputPluginName, m.TagList()), field.Value)
	}
}

// buildTexts constructs text metrics from a cua metric.
func (c *Circonus) buildTexts(defaultDest *cgm.CirconusMetrics, m cua.Metric) {
	fields := m.FieldList()
	for _, field := range fields {
		mn := m.Name()
		inputPluginName := ""
		mnp := strings.SplitN(mn, ".", 2)
		if len(mnp) == 2 {
			inputPluginName = mnp[0]
			mn = mnp[1]
		}
		mn += "." + field.Key

		dest := c.getMetricDest(defaultDest, inputPluginName)
		if dest == nil {
			// no default and no plugin specific
			return
		}

		if strings.HasSuffix(mn, "__value") {
			mn = mn[:len(mn)-7]
		}

		dest.SetTextWithTags(mn, c.convertTags(inputPluginName, m.TagList()), field.Value.(string))
	}
}

// buildHistogram constructs histogram metrics from a cua metric.
func (c *Circonus) buildHistogram(defaultDest *cgm.CirconusMetrics, m cua.Metric) {

	mn := m.Name()
	inputPluginName := ""
	mnp := strings.SplitN(mn, ".", 2)
	if len(mnp) == 2 {
		inputPluginName = mnp[0]
		mn = mnp[1]
	}

	dest := c.getMetricDest(defaultDest, inputPluginName)
	if dest == nil {
		// no default and no plugin specific
		return
	}

	if strings.HasSuffix(mn, "__value") {
		mn = mn[:len(mn)-7]
	}

	ctags := c.convertTags(inputPluginName, m.TagList())

	fields := m.FieldList()
	for _, f := range fields {
		v, err := strconv.ParseFloat(f.Key, 64)
		if err != nil {
			continue
		}

		dest.RecordCountForValueWithTags(mn, ctags, v, f.Value.(int64))
	}
}

// convertTags reformats cua tags to cgm tags
func (c *Circonus) convertTags(pluginName string, tags []*cua.Tag) cgm.Tags {
	var ctags cgm.Tags

	if len(tags) == 0 && pluginName == "" {
		return ctags
	}

	numTags := len(tags)
	if pluginName != "" {
		numTags++
	}

	ctags = make(cgm.Tags, numTags)
	if len(tags) > 0 {
		for i, t := range tags {
			if t.Key == "host" && c.CheckNamePrefix != "" {
				ctags[i] = cgm.Tag{Category: t.Key, Value: c.CheckNamePrefix}
				continue
			}
			ctags[i] = cgm.Tag{Category: t.Key, Value: t.Value}
		}
	}

	if pluginName != "" {
		ctags[len(tags)] = cgm.Tag{Category: "input_plugin", Value: pluginName}
	}

	return ctags
}
