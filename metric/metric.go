package metric

import (
	"fmt"
	"hash/fnv"
	"sort"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

type metric struct {
	tm                     time.Time
	name                   string
	originInstance         string
	origin                 string
	originCheckDipslayName string
	originCheckTarget      string
	originCheckTags        map[string]string
	fields                 []*cua.Field
	tags                   []*cua.Tag
	tp                     cua.ValueType
	aggregate              bool
}

func New(
	name string,
	tags map[string]string,
	fields map[string]interface{},
	tm time.Time,
	tp ...cua.ValueType,
) (cua.Metric, error) {
	var vtype cua.ValueType
	if len(tp) > 0 {
		vtype = tp[0]
	} else {
		vtype = cua.Untyped
	}

	m := &metric{
		name:   name,
		tags:   nil,
		fields: nil,
		tm:     tm,
		tp:     vtype,
	}

	if len(tags) > 0 {
		m.tags = make([]*cua.Tag, 0, len(tags))
		for k, v := range tags {
			m.tags = append(m.tags,
				&cua.Tag{Key: k, Value: v})
		}
		sort.Slice(m.tags, func(i, j int) bool { return m.tags[i].Key < m.tags[j].Key })
	}

	if len(fields) > 0 {
		m.fields = make([]*cua.Field, 0, len(fields))
		for k, v := range fields {
			v := convertField(v)
			if v == nil {
				continue
			}
			m.AddField(k, v)
		}
	}

	return m, nil
}

// FromMetric returns a deep copy of the metric with any tracking information
// removed.
func FromMetric(other cua.Metric) cua.Metric {
	m := &metric{
		name:                   other.Name(),
		tags:                   make([]*cua.Tag, len(other.TagList())),
		fields:                 make([]*cua.Field, len(other.FieldList())),
		tm:                     other.Time(),
		tp:                     other.Type(),
		aggregate:              other.IsAggregate(),
		origin:                 other.Origin(),
		originInstance:         other.OriginInstance(),
		originCheckTags:        make(map[string]string),
		originCheckTarget:      other.OriginCheckTarget(),
		originCheckDipslayName: other.OriginCheckDisplayName(),
	}

	for i, tag := range other.TagList() {
		m.tags[i] = &cua.Tag{Key: tag.Key, Value: tag.Value}
	}

	for i, field := range other.FieldList() {
		m.fields[i] = &cua.Field{Key: field.Key, Value: field.Value}
	}

	for k, v := range other.OriginCheckTags() {
		m.originCheckTags[k] = v
	}

	return m
}

func (m *metric) String() string {
	return fmt.Sprintf("%s %v %v %d", m.name, m.Tags(), m.Fields(), m.tm.UnixNano())
}

func (m *metric) Name() string {
	return m.name
}

func (m *metric) Tags() map[string]string {
	tags := make(map[string]string, len(m.tags))
	for _, tag := range m.tags {
		tags[tag.Key] = tag.Value
	}
	return tags
}

func (m *metric) TagList() []*cua.Tag {
	return m.tags
}

func (m *metric) Fields() map[string]interface{} {
	fields := make(map[string]interface{}, len(m.fields))
	for _, field := range m.fields {
		fields[field.Key] = field.Value
	}

	return fields
}

func (m *metric) FieldList() []*cua.Field {
	return m.fields
}

func (m *metric) Time() time.Time {
	return m.tm
}

func (m *metric) Type() cua.ValueType {
	return m.tp
}

func (m *metric) SetName(name string) {
	m.name = name
}

func (m *metric) AddPrefix(prefix string) {
	m.name = prefix + m.name
}

func (m *metric) AddSuffix(suffix string) {
	m.name += suffix
}

func (m *metric) AddTag(key, value string) {
	for i, tag := range m.tags {
		if key > tag.Key {
			continue
		}

		if key == tag.Key {
			tag.Value = value
			return
		}

		m.tags = append(m.tags, nil)
		copy(m.tags[i+1:], m.tags[i:])
		m.tags[i] = &cua.Tag{Key: key, Value: value}
		return
	}

	m.tags = append(m.tags, &cua.Tag{Key: key, Value: value})
}

func (m *metric) HasTag(key string) bool {
	for _, tag := range m.tags {
		if tag.Key == key {
			return true
		}
	}
	return false
}

func (m *metric) GetTag(key string) (string, bool) {
	for _, tag := range m.tags {
		if tag.Key == key {
			return tag.Value, true
		}
	}
	return "", false
}

func (m *metric) RemoveTag(key string) {
	for i, tag := range m.tags {
		if tag.Key == key {
			copy(m.tags[i:], m.tags[i+1:])
			m.tags[len(m.tags)-1] = nil
			m.tags = m.tags[:len(m.tags)-1]
			return
		}
	}
}

func (m *metric) AddField(key string, value interface{}) {
	for i, field := range m.fields {
		if key == field.Key {
			m.fields[i] = &cua.Field{Key: key, Value: convertField(value)}
			return
		}
	}
	m.fields = append(m.fields, &cua.Field{Key: key, Value: convertField(value)})
}

func (m *metric) HasField(key string) bool {
	for _, field := range m.fields {
		if field.Key == key {
			return true
		}
	}
	return false
}

func (m *metric) GetField(key string) (interface{}, bool) {
	for _, field := range m.fields {
		if field.Key == key {
			return field.Value, true
		}
	}
	return nil, false
}

func (m *metric) RemoveField(key string) {
	for i, field := range m.fields {
		if field.Key == key {
			copy(m.fields[i:], m.fields[i+1:])
			m.fields[len(m.fields)-1] = nil
			m.fields = m.fields[:len(m.fields)-1]
			return
		}
	}
}

func (m *metric) SetTime(t time.Time) {
	m.tm = t
}

func (m *metric) Copy() cua.Metric {
	m2 := &metric{
		name:                   m.name,
		tags:                   make([]*cua.Tag, len(m.tags)),
		fields:                 make([]*cua.Field, len(m.fields)),
		tm:                     m.tm,
		tp:                     m.tp,
		aggregate:              m.aggregate,
		origin:                 m.origin,
		originInstance:         m.originInstance,
		originCheckTags:        make(map[string]string),
		originCheckTarget:      m.originCheckTarget,
		originCheckDipslayName: m.originCheckDipslayName,
	}

	for i, tag := range m.tags {
		m2.tags[i] = &cua.Tag{Key: tag.Key, Value: tag.Value}
	}

	for i, field := range m.fields {
		m2.fields[i] = &cua.Field{Key: field.Key, Value: field.Value}
	}

	for k, v := range m.originCheckTags {
		m2.originCheckTags[k] = v
	}

	return m2
}

func (m *metric) SetAggregate(b bool) {
	m.aggregate = true
}

func (m *metric) IsAggregate() bool {
	return m.aggregate
}

func (m *metric) HashID() uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(m.name))
	_, _ = h.Write([]byte("\n"))
	for _, tag := range m.tags {
		_, _ = h.Write([]byte(tag.Key))
		_, _ = h.Write([]byte("\n"))
		_, _ = h.Write([]byte(tag.Value))
		_, _ = h.Write([]byte("\n"))
	}
	return h.Sum64()
}

func (m *metric) Accept() {
}

func (m *metric) Reject() {
}

func (m *metric) Drop() {
}

// Convert field to a supported type or nil if unconvertible
func convertField(v interface{}) interface{} {
	switch v := v.(type) {
	case float64:
		return v
	case int64:
		return v
	case string:
		return v
	case bool:
		return v
	case int:
		return int64(v)
	case uint:
		return uint64(v)
	case uint64:
		return v
	case []byte:
		return string(v)
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case uint32:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint8:
		return uint64(v)
	case float32:
		return float64(v)
	case *float64:
		if v != nil {
			return *v
		}
	case *int64:
		if v != nil {
			return *v
		}
	case *string:
		if v != nil {
			return *v
		}
	case *bool:
		if v != nil {
			return *v
		}
	case *int:
		if v != nil {
			return int64(*v)
		}
	case *uint:
		if v != nil {
			return uint64(*v)
		}
	case *uint64:
		if v != nil {
			return *v
		}
	case *[]byte:
		if v != nil {
			return string(*v)
		}
	case *int32:
		if v != nil {
			return int64(*v)
		}
	case *int16:
		if v != nil {
			return int64(*v)
		}
	case *int8:
		if v != nil {
			return int64(*v)
		}
	case *uint32:
		if v != nil {
			return uint64(*v)
		}
	case *uint16:
		if v != nil {
			return uint64(*v)
		}
	case *uint8:
		if v != nil {
			return uint64(*v)
		}
	case *float32:
		if v != nil {
			return float64(*v)
		}
	default:
		return nil
	}
	return nil
}

func (m *metric) Origin() string {
	return m.origin
}
func (m *metric) SetOrigin(origin string) {
	m.origin = origin
}

func (m *metric) OriginInstance() string {
	return m.originInstance
}
func (m *metric) SetOriginInstance(instanceID string) {
	m.originInstance = instanceID
}

func (m *metric) OriginCheckTags() map[string]string {
	ret := make(map[string]string)
	for k, v := range m.originCheckTags {
		ret[k] = v
	}
	return ret
}
func (m *metric) SetOriginCheckTags(checkTags map[string]string) {
	m.originCheckTags = make(map[string]string)
	for k, v := range checkTags {
		m.originCheckTags[k] = v
	}
}

func (m *metric) OriginCheckTarget() string {
	return m.originCheckTarget
}
func (m *metric) SetOriginCheckTarget(checkTarget string) {
	m.originCheckTarget = checkTarget
}

func (m *metric) OriginCheckDisplayName() string {
	return m.originCheckDipslayName
}
func (m *metric) SetOriginCheckDisplayName(checkDipslayName string) {
	m.originCheckDipslayName = checkDipslayName
}
