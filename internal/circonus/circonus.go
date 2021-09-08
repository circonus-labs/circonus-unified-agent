package circonus

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal/release"
	"github.com/circonus-labs/circonus-unified-agent/models"
	"github.com/circonus-labs/go-apiclient"
	"github.com/circonus-labs/go-trapcheck"
	"github.com/circonus-labs/go-trapmetrics"
	"github.com/shirou/gopsutil/v3/host"
)

var ch *Circonus

type Circonus struct {
	sync.Mutex
	logger           cua.Logger
	brokerTLSConfigs map[string]*tls.Config
	circCfg          *config.CirconusConfig
	apiCfg           *apiclient.Config
	brokerCIDrx      string
	globalTags       trapmetrics.Tags
	ready            bool
}

type MetricMeta struct {
	PluginID      string // plugin id or name (e.g. snmp, ping, etc.)
	InstanceID    string // plugin instance id (all plugins require an instance_id setting)
	MetricGroupID string // metric group id (some plugins produce multiple "metric groups")
	ProjectID     string // metric tag project_id (stackdriver_circonus)
}

type MetricDestConfig struct {
	DebugAPI     *bool   // allow override of api debugging per output
	TraceMetrics *string // allow override of metric tracing per output
	APIToken     string  // allow override of api token for a specific plugin (dm input or circonus output)
	Broker       string  // allow override of broker for a specific plugin (dm input or circonus output)
	Hostname     string  // allow override of hostname for a specific plugin (dm input or circonus output)
	MetricMeta   MetricMeta
}

// Logshim is for api and traps - it uses the info level and
// agent debug logging are controlled independently
type Logshim struct {
	logh     cua.Logger
	prefix   string
	debugAPI bool
}

func (l Logshim) Printf(fmt string, args ...interface{}) {
	if strings.Contains(fmt, "[DEBUG]") {
		// for retryablehttp (it only logs using Printf, and everything is DEBUG)
		if l.debugAPI {
			l.logh.Infof(l.prefix+": "+fmt, args...)
		}
	} else {
		l.logh.Infof(l.prefix+": "+fmt, args...)
	}
}
func (l Logshim) Debugf(fmt string, args ...interface{}) {
	l.logh.Debugf(l.prefix+": "+fmt, args...)
}
func (l Logshim) Infof(fmt string, args ...interface{}) {
	l.logh.Infof(l.prefix+": "+fmt, args...)
}
func (l Logshim) Warnf(fmt string, args ...interface{}) {
	l.logh.Warnf(l.prefix+": "+fmt, args...)
}
func (l Logshim) Errorf(fmt string, args ...interface{}) {
	l.logh.Errorf(l.prefix+": "+fmt, args...)
}

func Initialize(cfg *config.CirconusConfig, err error) error {
	if ch != nil {
		return nil // already initialized
	}
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("circonus metric destination management module: invalid circonus config (nil)")
	}

	c := &Circonus{
		circCfg:          cfg,
		brokerTLSConfigs: make(map[string]*tls.Config),
		globalTags:       make(trapmetrics.Tags, 0),
		brokerCIDrx:      `^/broker/[0-9]+$`,
	}

	if c.circCfg.APIToken == "" {
		return fmt.Errorf("circonus metric destination management module: unable to initialize, API Token is required")
	}

	if c.circCfg.APIApp == "" {
		c.circCfg.APIApp = "circonus-unified-agent"
	}

	c.apiCfg = &apiclient.Config{
		TokenKey:      c.circCfg.APIToken,
		TokenApp:      c.circCfg.APIApp,
		MaxRetries:    4,
		MinRetryDelay: "10s", // for race where api returns 500 but check is created,
		MaxRetryDelay: "20s", // if retry is to fast a duplicate check is created...
	}

	if c.circCfg.APIURL != "" {
		c.apiCfg.URL = c.circCfg.APIURL
	}

	if c.circCfg.APITLSCA != "" {
		cp := x509.NewCertPool()
		cert, err := os.ReadFile(c.circCfg.APITLSCA)
		if err != nil {
			return fmt.Errorf("circonus metric destination management module: unable to load api ca file (%s): %w", c.circCfg.APITLSCA, err)
		}
		if !cp.AppendCertsFromPEM(cert) {
			return fmt.Errorf("circonus metric destination management module: unable to parse api ca file (%s): %w", c.circCfg.APITLSCA, err)
		}
		c.apiCfg.CACert = cp
	}

	if c.circCfg.CacheConfigs && c.circCfg.CacheDir == "" {
		return fmt.Errorf("circonus metric destination management module: cache_configs on, cache_dir not set")
	}
	if c.circCfg.CacheConfigs && c.circCfg.CacheDir != "" {
		info, err := os.Stat(c.circCfg.CacheDir)
		if err != nil {
			return fmt.Errorf("circonus metric destination management module: cache_dir (%s): %w", c.circCfg.CacheDir, err)
		} else if !info.IsDir() {
			return fmt.Errorf("circonus metric destination management module: cache_dir (%s): not a directory", c.circCfg.CacheDir)
		}
	}

	// if len(c.circCfg.CheckSearchTags) == 0 {
	// 	_, an := filepath.Split(os.Args[0])
	// 	c.circCfg.CheckSearchTags = []string{"_service:" + an}
	// }

	if c.circCfg.Hostname == "" {
		hn, err := os.Hostname()
		if err != nil || hn == "" {
			hn = "unknown"
		}
		c.circCfg.Hostname = hn
	}

	c.logger = models.NewLogger("agent", "circ_metric_dest_mgr", "")

	c.ready = true

	ch = c

	return nil
}

func Ready() bool {
	if ch == nil {
		return false
	}
	return ch.ready
}

func AddGlobalTags(tags map[string]string) {
	if ch == nil {
		return
	}
	for k, v := range tags {
		if k != "" && v != "" {
			ch.globalTags = append(ch.globalTags, trapmetrics.Tag{Category: k, Value: v})
		}
	}
}

func GetGlobalTags() trapmetrics.Tags {
	return ch.globalTags
}

// getAPIClient returns a Circonus API client or an error
func getAPIClient(opts *MetricDestConfig) (*apiclient.API, error) {
	if ch == nil {
		return nil, fmt.Errorf("circonus metric destination management module: module not initialized")
	}
	if !ch.ready {
		return nil, fmt.Errorf("circonus metric destination management module: invalid agent circonus config")
	}

	cfg := *ch.apiCfg
	if opts != nil {
		// only option which may currently be overridden is the api key
		if opts.APIToken != "" {
			cfg.TokenKey = opts.APIToken
		}
	}

	client, err := apiclient.New(&cfg)
	if err != nil {
		return nil, fmt.Errorf("circonus metric destination management module: unable to initialize circonus api client: %w", err)
	}

	return client, nil
}

// createCheck retrieves, finds, or creates a Check bundle in Circonus and returns a trap check or an error
func createCheck(cfg *trapcheck.Config) (*trapcheck.TrapCheck, error) {
	if ch == nil {
		return nil, fmt.Errorf("circonus metric destination management module: module not initialized")
	}
	if !ch.ready {
		return nil, fmt.Errorf("circonus metric destination management module: invalid agent circonus config")
	}

	tc, err := trapcheck.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("circonus metric destination management module: creating trap check: %w", err)
	}
	return tc, nil
}

// createMetrics creates an instance of trap metrics and returns it or an error
func createMetrics(cfg *trapmetrics.Config) (*trapmetrics.TrapMetrics, error) {
	if ch == nil {
		return nil, fmt.Errorf("circonus metric destination management module: module not initialized")
	}
	if !ch.ready {
		return nil, fmt.Errorf("circonus metric destination management module: invalid agent circonus config")
	}

	tm, err := trapmetrics.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("circonus metric destination management module: creating trap metrics: %w", err)
	}
	return tm, nil
}

// NewMetricDestination will find/retrieve/create a new circonus check bundle and add it to a trap metrics instance to be
// used as a metric destination.
//  pluginID = id/name (e.g. inputs.cpu would be cpu, inputs.snmp would be snmp)
//  instanceID = instance_id setting from the config
//  metricGroupID = group of metrics from the plugin (some offer multiple)
//  hostname = used in the display name and target of the check
//  logger = an instance of cua logger (already configured for the plugin requesting the metric destination)
func NewMetricDestination(opts *MetricDestConfig, logger cua.Logger) (*trapmetrics.TrapMetrics, error) {
	if ch == nil {
		return nil, fmt.Errorf("circonus metric destination management module: module not initialized")
	}
	if !ch.ready {
		return nil, fmt.Errorf("circonus metric destination management module: invalid agent circonus config")
	}

	// serialize, don't want too many checks being created simultaneously - api rate limits, overwhelm broker, duplicate checks, etc.
	ch.Lock()
	defer ch.Unlock()

	pluginID := opts.MetricMeta.PluginID
	instanceID := opts.MetricMeta.InstanceID
	metricGroupID := opts.MetricMeta.MetricGroupID
	projectID := opts.MetricMeta.ProjectID
	hostname := opts.Hostname

	destKey := opts.MetricMeta.Key()

	if hostname == "" && ch.circCfg.Hostname != "" {
		hostname = ch.circCfg.Hostname
	}

	debugCheckSet := false
	debugAPI := ch.circCfg.DebugAPI
	traceMetrics := ch.circCfg.TraceMetrics

	if opts.DebugAPI != nil {
		debugAPI = *opts.DebugAPI
	}
	if opts.TraceMetrics != nil {
		traceMetrics = *opts.TraceMetrics
	}

	bundle := loadCheckConfig(destKey)
	if bundle != nil {
		// NOTE: api call debug won't be set on existing checks unless they are cached.
		//       submission debugging will work as the flags will be set after the
		//       check bundle is retrieved (found) via the api. so, the initial api calls
		//       to find the check would not be printed in the log.
		checkUUID := bundle.CheckUUIDs[0]
		if settings, found := ch.circCfg.DebugChecks[checkUUID]; found {
			options := strings.Split(settings, ",")
			if len(options) != 2 {
				ch.logger.Warnf("debug_checks invalid settings (%s): %s", checkUUID, settings)
			}
			traceMetrics = options[1]
			debug, err := strconv.ParseBool(options[0])
			if err != nil {
				ch.logger.Warnf("debug_checks invalid setting (%s) (debugapi:%s): %s", checkUUID, options[0], err)
			} else {
				debugAPI = debug
			}
			ch.logger.Infof("set debug:%t trace:%s on check %s", debugAPI, traceMetrics, checkUUID)
			debugCheckSet = true // don't try to parse the setting again later
		}
	}

	checkType := []string{"httptrap", "cua", pluginID}
	if metricGroupID != "" {
		checkType = append(checkType, metricGroupID)
	}
	checkType = append(checkType, runtime.GOOS)

	var checkDisplayName []string
	switch pluginID {
	case "stackdriver_circonus":
		checkDisplayName = []string{"GCP"}
		if instanceID != "" && instanceID != pluginID {
			checkDisplayName = append(checkDisplayName, instanceID)
		}
		if projectID != "" {
			checkDisplayName = append(checkDisplayName, projectID)
		}
		if metricGroupID != "" {
			checkDisplayName = append(checkDisplayName, gcpMetricGroupLookup(metricGroupID))
		}
	default:
		checkDisplayName = []string{hostname, pluginID}
		if instanceID != "" && instanceID != pluginID {
			checkDisplayName = append(checkDisplayName, instanceID)
		}
		checkDisplayName = append(checkDisplayName, "("+runtime.GOOS+")")
	}

	// tags used to SEARCH for a specific check
	searchTags := make([]string, 0)
	if len(ch.circCfg.CheckSearchTags) > 0 {
		for _, tag := range ch.circCfg.CheckSearchTags {
			if tag != "" {
				searchTags = append(searchTags, tag)
			}
		}
	}
	if !strings.Contains(strings.Join(searchTags, ","), "_instance_id") {
		searchTags = append(searchTags, "_instance_id:"+strings.ToLower(instanceID))
	}

	// additional tags to ADD to a check (metadata, DESCRIBE a check)
	checkTags := []string{
		"_plugin_id:" + pluginID,
		"_instance_id:" + strings.ToLower(instanceID),
		"_service:" + release.NAME,
	}
	if metricGroupID != "" {
		checkTags = append(checkTags, "_metric_group:"+metricGroupID)
	}

	if pluginID == "host" {
		checkTags = append(checkTags, getOSCheckTags()...)
	}

	checkTarget := hostname

	instanceLogger := &Logshim{
		logh:     logger,
		prefix:   destKey,
		debugAPI: debugAPI,
	}

	// API client
	circAPI, err := getAPIClient(opts)
	if err != nil {
		return nil, err
	}
	circAPI.Log = instanceLogger
	circAPI.Debug = debugAPI

	// Trap Check
	tc := &trapcheck.Config{
		Client:          circAPI,
		Logger:          instanceLogger,
		CheckSearchTags: searchTags,
		TraceMetrics:    traceMetrics,
	}

	var cc *apiclient.CheckBundle
	var tch *trapcheck.TrapCheck

	switch {
	case bundle != nil && ch.circCfg.CacheNoVerify: // use cached check bundle and don't verify by pulling from API again
		if tlscfg, ok := ch.brokerTLSConfigs[bundle.Brokers[0]]; ok {
			tc.SubmitTLSConfig = tlscfg.Clone()
		}
		var err error
		tch, err = trapcheck.NewFromCheckBundle(tc, bundle)
		if err != nil {
			return nil, err
		}
		if tc.SubmitTLSConfig == nil {
			t, err := tch.GetBrokerTLSConfig()
			if err != nil {
				return nil, fmt.Errorf("circonus metric destination management module: unable to get broker tls config: %w", err)
			}
			if t != nil {
				ch.brokerTLSConfigs[bundle.Brokers[0]] = t.Clone()
			} else {
				// note: err==nil and t==nil means public broker (api.circonus.com) or using http: as the schema
				ch.brokerTLSConfigs[bundle.Brokers[0]] = t
			}
		}
	case bundle != nil: // cached bundle, use the cid
		cc = &apiclient.CheckBundle{
			CID:  bundle.CID,
			Tags: checkTags,
		}
		if len(bundle.Brokers) > 0 {
			if tlscfg, ok := ch.brokerTLSConfigs[bundle.Brokers[0]]; ok {
				tc.SubmitTLSConfig = tlscfg.Clone()
			}
		}
	default: // find/create check bundle
		cc = &apiclient.CheckBundle{
			Type:        strings.Join(checkType, ":"),
			DisplayName: strings.Join(checkDisplayName, " "),
			Target:      checkTarget,
			Tags:        checkTags,
		}
		if opts.Broker != "" {
			cc.Brokers = []string{opts.Broker}
		} else if ch.circCfg.Broker != "" {
			cc.Brokers = []string{ch.circCfg.Broker}
		}
		if len(cc.Brokers) > 0 {
			// fixup config supplied broker CID if needed
			bid := cc.Brokers[0]
			if !strings.HasPrefix(bid, "/broker/") {
				bid = "/broker/" + bid
				matched, err := regexp.MatchString(ch.brokerCIDrx, bid)
				if err != nil {
					return nil, err
				}
				if !matched {
					return nil, fmt.Errorf("invalid broker cid (%s): %w", bid, err)
				}
				cc.Brokers[0] = bid
			}
			if tlscfg, ok := ch.brokerTLSConfigs[cc.Brokers[0]]; ok {
				tc.SubmitTLSConfig = tlscfg.Clone()
			}
		}
	}

	if tch == nil {
		var err error
		tc.CheckConfig = cc
		logger.Debug("find/create check using API")
		tch, err = createCheck(tc)
		if err != nil {
			return nil, err
		}
	}

	if bundle == nil { // it wasn't loaded from cache
		b, err := tch.GetCheckBundle()
		if err != nil {
			return nil, fmt.Errorf("circonus metric destination management module: unable to get check bundle: %w", err)
		}
		bundle = &b
		saveCheckConfig(destKey, bundle)
	}

	if pluginID == "host" {
		if b, err := updateCheckTags(circAPI, bundle, checkTags, logger); err != nil {
			logger.Warnf("circonus metric destination management moudle: updating check %s", err)
		} else if b != nil {
			saveCheckConfig(destKey, b)
		}
	}

	// if checks are going to a non-public trap
	// cache the brokerTLS to use for other checks
	// so that the api isn't hit for evevry check to pull the broker
	if _, ok := ch.brokerTLSConfigs[bundle.Brokers[0]]; !ok {
		t, err := tch.GetBrokerTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("circonus metric destination management module: unable to get broker tls config: %w", err)
		}
		if t != nil {
			ch.brokerTLSConfigs[bundle.Brokers[0]] = t.Clone()
		} else {
			// note: err==nil and t==nil means public broker (api.circonus.com) or using http: as the schema
			ch.brokerTLSConfigs[bundle.Brokers[0]] = t
		}
	}

	// Trap Metrics
	tm := &trapmetrics.Config{
		Trap:   tch,
		Logger: instanceLogger,
	}
	metrics, err := createMetrics(tm)
	if err != nil {
		return nil, err
	}

	if bundle != nil && !debugCheckSet {
		// non-nil bundle, and check specific debugging hasn't already been set
		// e.g. check wasn't cached, and was "found" during check initialization
		checkUUID := bundle.CheckUUIDs[0]
		if settings, found := ch.circCfg.DebugChecks[checkUUID]; found {
			options := strings.Split(settings, ",")
			if len(options) != 2 {
				ch.logger.Warnf("debug_checks invalid settings (%s): %s", checkUUID, settings)
			}
			traceMetrics = options[1]
			debug, err := strconv.ParseBool(options[0])
			if err != nil {
				ch.logger.Warnf("debug_checks invalid setting (%s) (debugapi:%s): %s", checkUUID, options[0], err)
			} else {
				debugAPI = debug
			}
			circAPI.Debug = debugAPI
			instanceLogger.debugAPI = debugAPI
			_, _ = tch.TraceMetrics(traceMetrics)
			ch.logger.Infof("set debug:%t trace:%s on check %s", debugAPI, traceMetrics, checkUUID)
		}
	}

	return metrics, nil
}

func getOSCheckTags() []string {
	ri := release.GetInfo()
	checkTags := []string{
		"_os:" + runtime.GOOS,
		"_agent:" + ri.Name + "-" + strings.ToLower(ri.Version),
	}

	hi, err := host.Info()
	if err != nil {
		return checkTags
	}
	// hi.OS is runtime.GOOS, which is added above
	// if hi.OS != "" {
	// 	checkTags = append(checkTags, "_os:"+hi.OS)
	// }
	if hi.Platform != "" {
		checkTags = append(checkTags, "_platform:"+strings.ToLower(hi.Platform))
	}
	if hi.PlatformFamily != "" {
		checkTags = append(checkTags, "_platform_family:"+strings.ToLower(hi.PlatformFamily))
	}
	if hi.PlatformVersion != "" {
		checkTags = append(checkTags, "_platform_version:"+strings.ToLower(hi.PlatformVersion))
	}
	if hi.KernelVersion != "" {
		checkTags = append(checkTags, "_kernel_version:"+strings.ToLower(hi.KernelVersion))
	}
	if hi.KernelArch != "" {
		checkTags = append(checkTags, "_kernel_arch:"+strings.ToLower(hi.KernelArch))
	}
	if hi.VirtualizationSystem != "" {
		checkTags = append(checkTags, "_virt_sys:"+strings.ToLower(hi.VirtualizationSystem))
	}
	if hi.VirtualizationRole != "" {
		checkTags = append(checkTags, "_virt_role:"+strings.ToLower(hi.VirtualizationRole))
	}

	return checkTags
}

func updateCheckTags(client *apiclient.API, bundle *apiclient.CheckBundle, tags []string, logger cua.Logger) (*apiclient.CheckBundle, error) {

	update := false
	for _, ostag := range tags {
		found := false
		ostagParts := strings.SplitN(ostag, ":", 2)
		for j, ctag := range bundle.Tags {
			if ostag == ctag {
				found = true
				break
			}

			ctagParts := strings.SplitN(ctag, ":", 2)
			if len(ostagParts) == len(ctagParts) {
				if ostagParts[0] == ctagParts[0] {
					if ostagParts[1] != ctagParts[1] {
						logger.Warnf("modifying tag: new: %v old: %v", ostagParts, ctagParts)
						bundle.Tags[j] = ostag
						update = true // but force update since we're modifying a tag
						found = true
						break
					}
				}

			}
		}
		if !found {
			logger.Warnf("adding missing tag: %s curr: %v", ostag, bundle.Tags)
			bundle.Tags = append(bundle.Tags, ostag)
			update = true
		}
	}

	if update {
		b, err := client.UpdateCheckBundle(bundle)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	return nil, nil
}

func (m MetricMeta) Key() string {
	return fmt.Sprintf("%s:%s:%s:%s",
		m.PluginID,
		m.InstanceID,
		m.MetricGroupID,
		m.ProjectID)
}
