package circonus

import (
	"time"

	"github.com/circonus-labs/go-trapmetrics"
)

// Contains helpers for direct metric input plugins

func AddMetricToDest(dest *trapmetrics.TrapMetrics, pluginID, metricGroup, metricName string, metricTags, staticInputTags map[string]string, value interface{}, ts time.Time) error {

	tags := ConvertTags(pluginID, metricGroup, metricTags, staticInputTags)

	switch v := value.(type) {
	case string:
		if err := dest.TextSet(metricName, tags, v, &ts); err != nil {
			return err
		}
	default:
		if err := dest.GaugeSet(metricName, tags, v, &ts); err != nil {
			return err
		}
	}

	return nil
}

func ConvertTags(pluginID, metricGroup string, tags, staticTags map[string]string) trapmetrics.Tags {
	var ctags trapmetrics.Tags

	if len(tags) == 0 && len(staticTags) == 0 && pluginID == "" {
		return ctags
	}
	if metricGroup == "" {
		metricGroup = pluginID
	}

	ctags = make(trapmetrics.Tags, 0)
	haveInputMetricGroup := false

	for key, val := range tags {
		if key == "input_metric_group" {
			haveInputMetricGroup = true
		}
		ctags = append(ctags, trapmetrics.Tag{Category: key, Value: val})
	}

	// add static input plugin tags
	for key, val := range staticTags {
		ctags = append(ctags, trapmetrics.Tag{Category: key, Value: val})
	}

	if pluginID != "" {
		ctags = append(ctags, trapmetrics.Tag{Category: "input_plugin", Value: pluginID})
	}
	if !haveInputMetricGroup && metricGroup != "" {
		if pluginID != "" && pluginID != metricGroup {
			ctags = append(ctags, trapmetrics.Tag{Category: "input_metric_group", Value: metricGroup})
		}
	}

	return ctags
}
