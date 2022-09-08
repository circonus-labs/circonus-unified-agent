package circonus

import (
	"context"
	"strings"
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

func UpdateCheckTags(ctx context.Context, dest *trapmetrics.TrapMetrics) error {

	b, err := dest.UpdateCheckTags(ctx)
	if err != nil {
		return err
	}

	// bundle will be nil if no updates were needed
	if b != nil {
		saveCheckConfig(dest.TrapID(), b)
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

// VerifyTags is used to ensure that a list of tags provided in a config are not blank
// and to lowercase them.
func VerifyTags(tags []string) ([]string, bool) {
	if len(tags) == 0 {
		return tags, true
	}

	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		result = append(result, strings.ToLower(tag))
	}

	return result, true
}

// MapToTags converts a map[string]string to []string
func MapToTags(mtags map[string]string) []string {
	if len(mtags) == 0 {
		return []string{}
	}

	tags := make([]string, 0, len(mtags))
	for k, v := range mtags {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)

		tag := ""
		switch {
		case k != "" && v == "": // just a category
			tag = k + ":"
		case k == "" && v != "": // just a value
			tag = ":" + v
		case k != "" && v != "": // category and value
			tag = k + ":" + v
		case k == "" && v == "": // empty, ignore
			continue
		}

		tags = append(tags, tag)
	}

	return tags
}
