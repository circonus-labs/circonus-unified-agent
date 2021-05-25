package circonus

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/maier/go-trapmetrics"
)

func (c *Circonus) metricProcessor(id int, metrics []cua.Metric) int64 {

	c.Log.Debugf("processor %d, received %d batches", id, len(metrics))

	start := time.Now()
	numMetrics := int64(0)
	for _, m := range metrics {
		switch m.Type() {
		case cua.Counter, cua.Gauge, cua.Summary:
			numMetrics += c.buildNumerics(m)
		case cua.Untyped:
			fields := m.FieldList()
			if s, ok := fields[0].Value.(string); ok {
				if strings.Contains(s, "H[") && strings.Contains(s, "]=") {
					numMetrics += c.buildHistogram(m)
				} else {
					numMetrics += c.buildTexts(m)
				}
			} else {
				numMetrics += c.buildNumerics(m)
			}
		case cua.Histogram:
			numMetrics += c.buildHistogram(m)
		case cua.CumulativeHistogram:
			numMetrics += c.buildCumulativeHistogram(m)
		default:
			c.Log.Warnf("processor %d, unknown type %T, ignoring", id, m)
		}
	}

	if agentDestination != nil {
		if err := agentDestination.metrics.GaugeAdd(metricVolume+"_batch", nil, numMetrics, &start); err != nil {
			c.Log.Warnf("adding gauge (%s): %s", metricVolume+"_batch", err)
		}
		agentDestination.queuedMetrics++
		if err := agentDestination.metrics.HistogramRecordValue(
			"cua_batch_queue_latency",
			trapmetrics.Tags{{Category: "units", Value: "microseconds"}},
			float64(time.Since(start).Nanoseconds()/int64(time.Microsecond))); err != nil {
			c.Log.Warnf("adding histogram sample (cua_batch_queue_latency): %s", err)
		}
		agentDestination.queuedMetrics++
	}

	c.Log.Debugf("processor %d, queued %d metrics for submission in %s", id, numMetrics, time.Since(start).String())

	sendStart := time.Now()
	var wg sync.WaitGroup
	c.RLock()
	ctx := context.Background()
	for _, dest := range c.metricDestinations {
		if dest.queuedMetrics == 0 {
			continue
		}
		wg.Add(1)
		go func(d *metricDestination) {
			defer wg.Done()
			subStart := time.Now()
			d.queuedMetrics = int64(0)
			result, err := d.metrics.Flush(ctx)
			if err != nil {
				c.Log.Warnf("submitting metrics (%s): %s", d.id, err)
				return
			}
			if agentDestination != nil {
				if err := agentDestination.metrics.HistogramRecordValue("cua_metrics_submitted", nil, float64(result.Stats)); err != nil {
					c.Log.Warnf("adding histogram sample (cua_metrics_submitted): %s", err)
				}
				agentDestination.queuedMetrics++
				if err := agentDestination.metrics.HistogramRecordValue("cua_submit_latency", trapmetrics.Tags{{Category: "units", Value: "milliseconds"}}, float64(time.Since(subStart).Milliseconds())); err != nil {
					c.Log.Warnf("adding histogram sample (cua_submit_latency): %s", err)
				}
				agentDestination.queuedMetrics++
			}
		}(dest)
	}

	wg.Wait()
	c.RUnlock()

	if agentDestination != nil {
		if err := agentDestination.metrics.HistogramRecordValue("cua_processor_latency", trapmetrics.Tags{{Category: "units", Value: "milliseconds"}}, float64(time.Since(start).Milliseconds())); err != nil {
			c.Log.Warnf("addindg histogram sample (cua_process_latency): %s", err)
		}
		agentDestination.queuedMetrics++
	}

	c.Log.Debugf("processor %d, submit sent metrics in %s", id, time.Since(sendStart))

	return numMetrics
}

// handleGeneric constructs text and numeric metrics from a cua metric
// Note: for certain cua metric types the actual fields may be either text OR numeric...
func (c *Circonus) handleGeneric(m cua.Metric) int64 {
	dest := c.getMetricDestination(m)
	if dest == nil {
		c.Log.Warnf("no metric destination found for metric (%#v)", m)
		return 0
	}
	numMetrics := int64(0)
	tags := c.convertTags(m)
	batchTS := m.Time()

	for _, field := range m.FieldList() {
		mn := strings.TrimSuffix(field.Key, "__value")
		if c.DebugMetrics {
			c.Log.Infof("%s %v %v %T\n", mn, tags.String(), field.Value, field.Value)
		}
		switch v := field.Value.(type) {
		case string:
			if err := dest.metrics.TextSet(mn, tags, v, &batchTS); err != nil {
				c.Log.Warnf("setting text (%s %s): %s", mn, tags.String(), err)
			}
		default: // treat it as a numeric
			if err := dest.metrics.GaugeSet(mn, tags, v, &batchTS); err != nil {
				c.Log.Warnf("setting gauge (%s %s): %s", mn, tags.String(), err)
			}
		}
		numMetrics++
	}

	dest.queuedMetrics += numMetrics

	return numMetrics
}

// buildNumerics constructs numeric metrics from a cua metric.
func (c *Circonus) buildNumerics(m cua.Metric) int64 {
	return c.handleGeneric(m)
}

// buildTexts constructs text metrics from a cua metric.
func (c *Circonus) buildTexts(m cua.Metric) int64 {
	return c.handleGeneric(m)
}

// buildHistogram constructs histogram metrics from a cua metric.
func (c *Circonus) buildHistogram(m cua.Metric) int64 {
	dest := c.getMetricDestination(m)
	if dest == nil {
		c.Log.Warnf("no metric destination found for metric (%#v)", m)
		return 0
	}

	numMetrics := int64(0)
	mn := strings.TrimSuffix(m.Name(), "__value")
	tags := c.convertTags(m)

	for _, field := range m.FieldList() {
		v, err := strconv.ParseFloat(field.Key, 64)
		if err != nil {
			c.Log.Errorf("cannot parse histogram (%s) field.key (%s) as float: %s\n", mn, field.Key, err)
			continue
		}
		if c.DebugMetrics {
			c.Log.Infof("%s %v v:%v vt%T n:%v nT:%T\n", mn, tags, v, v, field.Value, field.Value)
		}

		if err := dest.metrics.HistogramRecordCountForValue(mn, tags, field.Value.(int64), v); err != nil {
			c.Log.Warnf("setting gauge (%s %s): %s", mn, tags.String(), err)
		}
		numMetrics++
	}

	dest.queuedMetrics += numMetrics

	return numMetrics
}

// buildCumulativeHistogram constructs cumulative histogram metrics from a cua metric.
func (c *Circonus) buildCumulativeHistogram(m cua.Metric) int64 {
	dest := c.getMetricDestination(m)
	if dest == nil {
		c.Log.Warnf("no metric destination found for metric (%#v)", m)
		return 0
	}

	numMetrics := int64(0)
	mn := strings.TrimSuffix(m.Name(), "__value")
	tags := c.convertTags(m)

	for _, field := range m.FieldList() {
		v, err := strconv.ParseFloat(field.Key, 64)
		if err != nil {
			c.Log.Errorf("cannot parse histogram (%s) field.key (%s) as float: %s\n", mn, field.Key, err)
			continue
		}
		if c.DebugMetrics {
			c.Log.Infof("%s %v v:%v vt%T n:%v nT:%T\n", mn, tags, v, v, field.Value, field.Value)
		}

		if err := dest.metrics.CumulativeHistogramRecordCountForValue(mn, tags, field.Value.(int64), v); err != nil {
			c.Log.Warnf("setting gauge (%s %s): %s", mn, tags.String(), err)
		}

		numMetrics++
	}

	dest.queuedMetrics += numMetrics

	return numMetrics
}

// convertTags reformats cua tags to cgm tags
func (c *Circonus) convertTags(m cua.Metric) trapmetrics.Tags { //nolint:unparam
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

func (c *Circonus) getMetricGroupTag(m cua.Metric) string {
	for _, t := range m.TagList() {
		if t.Key == "input_metric_group" {
			return t.Value
		}
	}
	if m.Name() != "" && m.Name() != m.Origin() {
		// what the plugin identifies a subgroup of metrics as, some have multiple names
		// e.g. internal, smart, aws, etc.
		return m.Name()
	}
	return ""
}
