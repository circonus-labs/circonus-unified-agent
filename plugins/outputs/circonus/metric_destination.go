package circonus

import (
	"os"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	circmgr "github.com/circonus-labs/circonus-unified-agent/internal/circonus"
	"github.com/circonus-labs/go-trapmetrics"
)

type metricDestination struct {
	metrics       *trapmetrics.TrapMetrics
	id            string
	queuedMetrics int64
}

// getMetricDestination returns a destination for the plugin identified by a plugin and plugin instance id
func (c *Circonus) getMetricDestination(m cua.Metric) *metricDestination {
	pluginID := m.Origin()
	instanceID := m.OriginInstance()

	// default - used in two cases:
	// 1. plugin cannot be identified
	// 2. user as enabled one_check
	var defaultDest *metricDestination
	if defaultDestination != nil {
		defaultDest = defaultDestination
	}

	if c.OneCheck || pluginID == "" {
		return defaultDest
	}

	if config.IsDefaultInstanceID(instanceID) {
		// host metrics - the "default" plugins which are enabled by default
		// but can be controlled via the (ENABLE_DEFAULT_PLUGINS env var
		// any value other than "false" will enable the default plugins)
		if config.IsDefaultPlugin(pluginID) {
			hostDest := defaultDest
			if config.DefaultPluginsEnabled() {
				if hostDestination != nil {
					hostDest = hostDestination
				}
			}
			return hostDest
		}
		// agent metrics - metrics the agent emits about itself - always enabled
		if config.IsAgentPlugin(pluginID) {
			agentDest := defaultDest
			if agentDestination != nil {
				agentDest = agentDestination
			}
			return agentDest
		}
	}

	metricGroupID := ""
	if pluginID == "stackdriver_circonus" {
		metricGroupID = c.getMetricGroupTag(m)
		if metricGroupID != "" {
			parts := strings.SplitN(metricGroupID, "/", 2)
			if len(parts) > 0 {
				metricGroupID = parts[0]
			}
		}
	}

	destKey := circmgr.MakeDestinationKey(pluginID, instanceID, metricGroupID)

	c.RLock()
	d, found := c.metricDestinations[destKey]
	c.RUnlock()
	if found {
		return d
	}

	if err := c.initMetricDestination(pluginID, instanceID, metricGroupID); err != nil {
		c.Log.Errorf("error initializing metric destination: %s", err)
		os.Exit(1) //nolint:gocritic
	}

	c.RLock()
	d, found = c.metricDestinations[destKey]
	c.RUnlock()
	if found {
		return d
	}

	return defaultDest
}

func (c *Circonus) initMetricDestination(pluginID, instanceID, metricGroupID string) error {
	c.Lock()
	defer c.Unlock()

	opts := circmgr.MetricDestConfig{
		PluginID:        pluginID,
		InstanceID:      instanceID,
		MetricGroupID:   metricGroupID,
		APIToken:        c.APIToken,
		Broker:          c.Broker,
		CheckNamePrefix: c.CheckNamePrefix,
		DebugAPI:        c.DebugAPI,
		TraceMetrics:    c.TraceMetrics,
	}

	dest, err := circmgr.NewMetricDestination(&opts, c.Log)
	if err != nil {
		return err
	}

	destKey := circmgr.MakeDestinationKey(pluginID, instanceID, metricGroupID)

	c.metricDestinations[destKey] = &metricDestination{
		metrics: dest,
		id:      pluginID,
	}

	return nil
}
