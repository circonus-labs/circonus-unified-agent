package json

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
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
	m := s.createObject(metric)
	serialized, err := json.Marshal(m)
	if err != nil {
		return []byte{}, fmt.Errorf("json marshal: %w", err)
	}
	serialized = append(serialized, '\n')

	return serialized, nil
}

func (s *Serializer) SerializeBatch(metrics []cua.Metric) ([]byte, error) {
	objects := make([]interface{}, 0, len(metrics))
	for _, metric := range metrics {
		m := s.createObject(metric)
		objects = append(objects, m)
	}

	obj := map[string]interface{}{
		"metrics": objects,
	}

	serialized, err := json.Marshal(obj)
	if err != nil {
		return []byte{}, fmt.Errorf("json marshal: %w", err)
	}
	return serialized, nil
}

func (s *Serializer) createObject(metric cua.Metric) map[string]interface{} {
	m := make(map[string]interface{}, 4)

	tags := make(map[string]string, len(metric.TagList()))
	for _, tag := range metric.TagList() {
		tags[tag.Key] = tag.Value
	}
	m["tags"] = tags

	fields := make(map[string]interface{}, len(metric.FieldList()))
	for _, field := range metric.FieldList() {
		switch fv := field.Value.(type) {
		case float64:
			// JSON does not support these special values
			if math.IsNaN(fv) || math.IsInf(fv, 0) { //nolint:staticcheck
				continue
			}
		default:
			fields[field.Key] = field.Value
		}
	}
	m["fields"] = fields

	m["name"] = metric.Name()
	m["timestamp"] = metric.Time().UnixNano() / int64(s.TimestampUnits)
	return m
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
