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

	if config.IsDefaultInstanceID(instanceID) {
		// host metrics - the "default" plugins which are enabled by default
		// but can be controlled via the (ENABLE_DEFAULT_PLUGINS env var
		// any value other than "false" will enable the default plugins)
		if config.IsDefaultPlugin(pluginID) {
			var hostDest *metricDestination
			if config.DefaultPluginsEnabled() {
				if c.hostDestination != nil {
					hostDest = c.hostDestination
				}
			}
			return hostDest
		}
		// agent metrics - metrics the agent emits about itself - always enabled
		if config.IsAgentPlugin(pluginID) {
			var agentDest *metricDestination
			if c.agentDestination != nil {
				agentDest = c.agentDestination
			}
			return agentDest
		}
	}

	metricGroupID := ""
	projectID := ""
	if pluginID == "stackdriver_circonus" {
		metricGroupID = c.getMetricGroupTag(m)
		if metricGroupID != "" {
			parts := strings.SplitN(metricGroupID, "/", 2)
			if len(parts) > 0 {
				metricGroupID = parts[0]
			}
		}
		projectID = c.getMetricProjectTag(m)
	}

	metricMeta := circmgr.MetricMeta{
		PluginID:      pluginID,
		InstanceID:    instanceID,
		MetricGroupID: metricGroupID,
		ProjectID:     projectID,
	}
	destKey := metricMeta.Key()

	c.RLock()
	d, found := c.metricDestinations[destKey]
	c.RUnlock()
	if found {
		return d
	}

	if err := c.initMetricDestination(metricMeta, m.OriginCheckTags(), m.OriginCheckTarget(), m.OriginCheckDisplayName()); err != nil {
		c.Log.Errorf("error initializing metric destination: %s", err)
		os.Exit(1) //nolint:gocritic
	}

	c.RLock()
	d, found = c.metricDestinations[destKey]
	c.RUnlock()
	if found {
		return d
	}

	return nil
}

func (c *Circonus) initMetricDestination(metricMeta circmgr.MetricMeta, checkTags map[string]string, checkTarget, checkDisplayName string) error {
	c.Lock()
	defer c.Unlock()

	opts := circmgr.MetricDestConfig{
		MetricMeta:       metricMeta,
		APIToken:         c.APIToken,
		Broker:           c.Broker,
		DebugAPI:         c.DebugAPI,
		TraceMetrics:     c.TraceMetrics,
		CheckTarget:      checkTarget,
		CheckTags:        checkTags,
		CheckDisplayName: checkDisplayName,
	}

	dest, err := circmgr.NewMetricDestination(&opts, c.Log)
	if err != nil {
		return err
	}

	destKey := metricMeta.Key()

	c.metricDestinations[destKey] = &metricDestination{
		metrics: dest,
		id:      metricMeta.PluginID,
	}

	return nil
}
