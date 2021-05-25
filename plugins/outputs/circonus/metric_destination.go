package circonus

import (
	"os"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	circmgr "github.com/circonus-labs/circonus-unified-agent/internal/circonus"
	"github.com/maier/go-trapmetrics"
)

type metricDestination struct {
	metrics       *trapmetrics.TrapMetrics
	id            string
	queuedMetrics int64
}

// getMetricDestination returns a destination for the plugin identified by a plugin and plugin instance id
func (c *Circonus) getMetricDestination(m cua.Metric) *metricDestination {
	plugin := m.Origin()
	instanceID := m.OriginInstance()

	// default - used in two cases:
	// 1. plugin cannot be identified
	// 2. user as enabled one_check
	var defaultDest *metricDestination
	if defaultDestination != nil {
		defaultDest = defaultDestination
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
				if hostDestination != nil {
					hostDest = hostDestination
				}
			}
			return hostDest
		}
		// agent metrics - metrics the agent emits about itself - always enabled
		if config.IsAgentPlugin(plugin) {
			agentDest := defaultDest
			if agentDestination != nil {
				agentDest = agentDestination
			}
			return agentDest
		}
	}

	id := plugin
	if instanceID != "" {
		id += ":" + instanceID
	}
	metricGroup := ""
	if plugin == "stackdriver_circonus" {
		metricGroup = c.getMetricGroupTag(m)
		if metricGroup != "" {
			parts := strings.SplitN(metricGroup, "/", 2)
			if len(parts) > 0 {
				id += ":" + parts[0]
				metricGroup = parts[0]
			} else {
				id += ":" + metricGroup
			}
		}
	}

	c.RLock()
	d, found := c.metricDestinations[id]
	c.RUnlock()

	if found {
		return d
	}

	if err := c.initMetricDestination(id, plugin+" "+instanceID+" "+metricGroup, instanceID); err == nil {
		if d, ok := c.metricDestinations[id]; ok {
			return d
		}
	} else {
		c.Log.Errorf("error initializing metric destination: %s", err)
		os.Exit(1) //nolint:gocritic
	}

	return defaultDest
}

func (c *Circonus) initMetricDestination(id, name, instanceID string) error {
	c.Lock()
	defer c.Unlock()

	plugID := id
	if id == "*" {
		plugID = "default"
		name = "default"
	}

	dest, err := circmgr.NewMetricDestination(plugID, name, instanceID, c.CheckNamePrefix, c.Log)
	if err != nil {
		return err
	}

	c.metricDestinations[id] = &metricDestination{
		metrics: dest,
		id:      plugID,
	}

	return nil
}
