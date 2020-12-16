package metric

import (
	"hash/fnv"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

// NewSeriesGrouper returns a type that can be used to group fields by series
// and time, so that fields which share these values will be combined into a
// single cua.Metric.
//
// This is useful to build cua.Metric's when all fields for a series are
// not available at once.
//
// ex:
// - cpu,host=localhost usage_time=42
// - cpu,host=localhost idle_time=42
// + cpu,host=localhost idle_time=42,usage_time=42
func NewSeriesGrouper() *SeriesGrouper {
	return &SeriesGrouper{
		metrics: make(map[uint64]cua.Metric),
		ordered: []cua.Metric{},
	}
}

type SeriesGrouper struct {
	metrics map[uint64]cua.Metric
	ordered []cua.Metric
}

// Add adds a field key and value to the series.
func (g *SeriesGrouper) Add(
	measurement string,
	tags map[string]string,
	tm time.Time,
	field string,
	fieldValue interface{},
) error {
	var err error
	id := groupID(measurement, tags, tm)
	metric := g.metrics[id]
	if metric == nil {
		metric, err = New(measurement, tags, map[string]interface{}{field: fieldValue}, tm)
		if err != nil {
			return err
		}
		g.metrics[id] = metric
		g.ordered = append(g.ordered, metric)
	} else {
		metric.AddField(field, fieldValue)
	}
	return nil
}

// Metrics returns the metrics grouped by series and time.
func (g *SeriesGrouper) Metrics() []cua.Metric {
	return g.ordered
}

func groupID(measurement string, tags map[string]string, tm time.Time) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(measurement))
	_, _ = h.Write([]byte("\n"))

	taglist := make([]*cua.Tag, 0, len(tags))
	for k, v := range tags {
		taglist = append(taglist,
			&cua.Tag{Key: k, Value: v})
	}
	sort.Slice(taglist, func(i, j int) bool { return taglist[i].Key < taglist[j].Key })
	for _, tag := range taglist {
		_, _ = h.Write([]byte(tag.Key))
		_, _ = h.Write([]byte("\n"))
		_, _ = h.Write([]byte(tag.Value))
		_, _ = h.Write([]byte("\n"))
	}
	_, _ = h.Write([]byte("\n"))

	_, _ = io.WriteString(h, strconv.FormatInt(tm.UnixNano(), 10))
	return h.Sum64()
}
