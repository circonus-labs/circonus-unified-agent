package circonus

import (
	"bytes"
	"fmt"
	"math"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/go-trapmetrics"
)

type Serializer struct {
	TimestampUnits time.Duration
}

func NewSerializer(timestampUnits time.Duration) (*Serializer, error) {
	s := &Serializer{
		TimestampUnits: truncateDuration(timestampUnits),
	}
	return s, nil
}

func (s *Serializer) Serialize(metric cua.Metric) ([]byte, error) {
	return s.SerializeBatch([]cua.Metric{metric})
}

func (s *Serializer) SerializeBatch(metrics []cua.Metric) ([]byte, error) {
	var buf bytes.Buffer

	for _, metric := range metrics {
		tags := s.convertTags(metric)
		for _, field := range metric.FieldList() {
			mt := ""
			switch fv := field.Value.(type) {
			case float64:
				// JSON does not support these special values
				if math.IsNaN(fv) || math.IsInf(fv, 0) { //nolint:staticcheck
					continue
				}
				mt = "n"
			case string:
				mt = "s"
			default:
				mt = "L"
			}

			_, _ = buf.WriteString(fmt.Sprintf(
				"{%q: {\"_value\": %v, \"_type\": %q, \"_ts\": %d}}\n",
				field.Key+"|ST["+tags.String()+"]",
				field.Value,
				mt,
				metric.Time().UnixNano()/int64(s.TimestampUnits),
			))
		}
	}

	return buf.Bytes(), nil
}

func truncateDuration(units time.Duration) time.Duration {
	// Default precision is 1s
	if units <= 0 {
		return time.Second
	}

	// Search for the power of ten less than the duration
	d := time.Nanosecond
	for {
		if d*10 > units {
			return d
		}
		d *= 10
	}
}

// convertTags reformats cua tags to cgm tags
func (s *Serializer) convertTags(m cua.Metric) trapmetrics.Tags { //nolint:unparam
	var ctags trapmetrics.Tags

	tags := m.TagList()

	if len(tags) == 0 && m.Origin() == "" {
		return ctags
	}

	ctags = make(trapmetrics.Tags, 0)

	haveInputMetricGroup := false

	if len(tags) > 0 {
		for _, t := range tags {
			if t.Key == "input_metric_group" {
				haveInputMetricGroup = true
			}
			ctags = append(ctags, trapmetrics.Tag{Category: t.Key, Value: t.Value})
		}
	}

	if m.Origin() != "" {
		// from config file `inputs.*`, the part after period
		ctags = append(ctags, trapmetrics.Tag{Category: "input_plugin", Value: m.Origin()})
	}
	if !haveInputMetricGroup {
		if m.Name() != "" && m.Name() != m.Origin() {
			// what the plugin identifies a subgroup of metrics as, some have multiple names
			// e.g. internal, smart, aws, etc.
			ctags = append(ctags, trapmetrics.Tag{Category: "input_metric_group", Value: m.Name()})
		}
	}

	return ctags
}
