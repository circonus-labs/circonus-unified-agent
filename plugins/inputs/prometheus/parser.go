package prometheus

// Parser inspired from
// https://github.com/prometheus/prom2json/blob/master/main.go

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/metric"
	"github.com/matttproud/golang_protobuf_extensions/pbutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// Parse returns a slice of Metrics from a text representation of a
// metrics
func ParseV2(buf []byte, header http.Header) ([]cua.Metric, error) {
	var metrics []cua.Metric
	var parser expfmt.TextParser
	// parse even if the buffer begins with a newline
	buf = bytes.TrimPrefix(buf, []byte("\n"))
	// Read raw data
	buffer := bytes.NewBuffer(buf)
	reader := bufio.NewReader(buffer)

	mediatype, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	// Prepare output
	metricFamilies := make(map[string]*dto.MetricFamily)

	if err == nil && mediatype == "application/vnd.google.protobuf" &&
		params["encoding"] == "delimited" &&
		params["proto"] == "io.prometheus.client.MetricFamily" {
		for {
			mf := &dto.MetricFamily{}
			if _, ierr := pbutil.ReadDelimited(reader, mf); ierr != nil {
				if errors.Is(ierr, io.EOF) {
					break
				}
				return nil, fmt.Errorf("reading metric family protocol buffer failed: %w", ierr)
			}
			metricFamilies[mf.GetName()] = mf
		}
	} else {
		metricFamilies, err = parser.TextToMetricFamilies(reader)
		if err != nil {
			return nil, fmt.Errorf("reading text format failed: %w", err)
		}
	}

	// make sure all metrics have a consistent timestamp so that metrics don't straddle two different seconds
	now := time.Now()
	// read metrics
	for metricName, mf := range metricFamilies {
		for _, m := range mf.Metric {
			// reading tags
			tags := makeLabels(m)

			switch mf.GetType() {
			case dto.MetricType_SUMMARY:
				// summary metric
				agentMetrics := makeQuantilesV2(m, tags, metricName, mf.GetType(), now)
				metrics = append(metrics, agentMetrics...)
			case dto.MetricType_HISTOGRAM:
				// histogram metric
				agentMetrics := makeBucketsV2(m, tags, metricName, mf.GetType(), now)
				metrics = append(metrics, agentMetrics...)
			default:
				// standard metric
				// reading fields
				fields := getNameAndValueV2(m, metricName)
				// converting to circonus metric
				if len(fields) > 0 {
					var t time.Time
					if m.TimestampMs != nil && *m.TimestampMs > 0 {
						t = time.Unix(0, *m.TimestampMs*1000000)
					} else {
						t = now
					}
					metric, err := metric.New("prometheus", tags, fields, t, valueType(mf.GetType()))
					if err == nil {
						metrics = append(metrics, metric)
					}
				}
			}
		}
	}

	return metrics, fmt.Errorf("metric new: %w", err)
}

// Get Quantiles for summary metric & Buckets for histogram
func makeQuantilesV2(m *dto.Metric, tags map[string]string, metricName string, metricType dto.MetricType, now time.Time) []cua.Metric {
	var metrics []cua.Metric
	fields := make(map[string]interface{})
	var t time.Time
	if m.TimestampMs != nil && *m.TimestampMs > 0 {
		t = time.Unix(0, *m.TimestampMs*1000000)
	} else {
		t = now
	}
	fields[metricName+"_count"] = float64(m.GetSummary().GetSampleCount())
	fields[metricName+"_sum"] = m.GetSummary().GetSampleSum()
	met, err := metric.New("prometheus", tags, fields, t, valueType(metricType))
	if err == nil {
		metrics = append(metrics, met)
	}

	for _, q := range m.GetSummary().Quantile {
		newTags := tags
		fields = make(map[string]interface{})

		newTags["quantile"] = fmt.Sprint(q.GetQuantile())
		fields[metricName] = q.GetValue()

		quantileMetric, err := metric.New("prometheus", newTags, fields, t, valueType(metricType))
		if err == nil {
			metrics = append(metrics, quantileMetric)
		}
	}
	return metrics
}

// Get Buckets  from histogram metric
func makeBucketsV2(m *dto.Metric, tags map[string]string, metricName string, metricType dto.MetricType, now time.Time) []cua.Metric {
	var metrics []cua.Metric
	fields := make(map[string]interface{})
	var t time.Time
	if m.TimestampMs != nil && *m.TimestampMs > 0 {
		t = time.Unix(0, *m.TimestampMs*1000000)
	} else {
		t = now
	}
	fields[metricName+"_count"] = float64(m.GetHistogram().GetSampleCount())
	fields[metricName+"_sum"] = m.GetHistogram().GetSampleSum()

	met, err := metric.New("prometheus", tags, fields, t, valueType(metricType))
	if err == nil {
		metrics = append(metrics, met)
	}

	for _, b := range m.GetHistogram().Bucket {
		newTags := tags
		fields = make(map[string]interface{})
		newTags["le"] = fmt.Sprint(b.GetUpperBound())
		fields[metricName+"_bucket"] = float64(b.GetCumulativeCount())

		histogramMetric, err := metric.New("prometheus", newTags, fields, t, valueType(metricType))
		if err == nil {
			metrics = append(metrics, histogramMetric)
		}
	}
	return metrics
}

// Parse returns a slice of Metrics from a text representation of a
// metrics
func Parse(buf []byte, header http.Header) ([]cua.Metric, error) {
	var metrics []cua.Metric
	var parser expfmt.TextParser
	// parse even if the buffer begins with a newline
	buf = bytes.TrimPrefix(buf, []byte("\n"))
	// Read raw data
	buffer := bytes.NewBuffer(buf)
	reader := bufio.NewReader(buffer)

	mediatype, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	// Prepare output
	metricFamilies := make(map[string]*dto.MetricFamily)

	if err == nil && mediatype == "application/vnd.google.protobuf" &&
		params["encoding"] == "delimited" &&
		params["proto"] == "io.prometheus.client.MetricFamily" {
		for {
			mf := &dto.MetricFamily{}
			if _, ierr := pbutil.ReadDelimited(reader, mf); ierr != nil {
				if errors.Is(ierr, io.EOF) {
					break
				}
				return nil, fmt.Errorf("reading metric family protocol buffer failed: %w", ierr)
			}
			metricFamilies[mf.GetName()] = mf
		}
	} else {
		metricFamilies, err = parser.TextToMetricFamilies(reader)
		if err != nil {
			return nil, fmt.Errorf("reading text format failed: %w", err)
		}
	}

	// make sure all metrics have a consistent timestamp so that metrics don't straddle two different seconds
	now := time.Now()
	// read metrics
	for metricName, mf := range metricFamilies {
		for _, m := range mf.Metric {
			// reading tags
			tags := makeLabels(m)
			// reading fields
			var fields map[string]interface{}
			switch mf.GetType() {
			case dto.MetricType_SUMMARY:
				// summary metric
				fields = makeQuantiles(m)
				fields["count"] = float64(m.GetSummary().GetSampleCount())
				fields["sum"] = m.GetSummary().GetSampleSum()
			case dto.MetricType_HISTOGRAM:
				// histogram metric
				fields = makeBuckets(m)
				fields["count"] = float64(m.GetHistogram().GetSampleCount())
				fields["sum"] = m.GetHistogram().GetSampleSum()
			default:
				// standard metric
				fields = getNameAndValue(m)
			}
			// converting to circonus metric
			if len(fields) > 0 {
				var t time.Time
				if m.TimestampMs != nil && *m.TimestampMs > 0 {
					t = time.Unix(0, *m.TimestampMs*1000000)
				} else {
					t = now
				}
				metric, err := metric.New(metricName, tags, fields, t, valueType(mf.GetType()))
				if err == nil {
					metrics = append(metrics, metric)
				}
			}
		}
	}

	return metrics, fmt.Errorf("metric new: %w", err)
}

func valueType(mt dto.MetricType) cua.ValueType {
	switch mt {
	case dto.MetricType_COUNTER:
		return cua.Counter
	case dto.MetricType_GAUGE:
		return cua.Gauge
	case dto.MetricType_SUMMARY:
		return cua.Summary
	case dto.MetricType_HISTOGRAM:
		return cua.Histogram
	default:
		return cua.Untyped
	}
}

// Get Quantiles from summary metric
func makeQuantiles(m *dto.Metric) map[string]interface{} {
	fields := make(map[string]interface{})
	for _, q := range m.GetSummary().Quantile {
		if !math.IsNaN(q.GetValue()) {
			fields[fmt.Sprint(q.GetQuantile())] = q.GetValue()
		}
	}
	return fields
}

// Get Buckets  from histogram metric
func makeBuckets(m *dto.Metric) map[string]interface{} {
	fields := make(map[string]interface{})
	for _, b := range m.GetHistogram().Bucket {
		fields[fmt.Sprint(b.GetUpperBound())] = float64(b.GetCumulativeCount())
	}
	return fields
}

// Get labels from metric
func makeLabels(m *dto.Metric) map[string]string {
	result := map[string]string{}
	for _, lp := range m.Label {
		result[lp.GetName()] = lp.GetValue()
	}
	return result
}

// Get name and value from metric
func getNameAndValue(m *dto.Metric) map[string]interface{} {
	fields := make(map[string]interface{})
	switch {
	case m.Gauge != nil:
		if !math.IsNaN(m.GetGauge().GetValue()) {
			fields["gauge"] = m.GetGauge().GetValue()
		}
	case m.Counter != nil:
		if !math.IsNaN(m.GetCounter().GetValue()) {
			fields["counter"] = m.GetCounter().GetValue()
		}
	case m.Untyped != nil:
		if !math.IsNaN(m.GetUntyped().GetValue()) {
			fields["value"] = m.GetUntyped().GetValue()
		}
	}
	return fields
}

// Get name and value from metric
func getNameAndValueV2(m *dto.Metric, metricName string) map[string]interface{} {
	fields := make(map[string]interface{})
	switch {
	case m.Gauge != nil:
		if !math.IsNaN(m.GetGauge().GetValue()) {
			fields[metricName] = m.GetGauge().GetValue()
		}
	case m.Counter != nil:
		if !math.IsNaN(m.GetCounter().GetValue()) {
			fields[metricName] = m.GetCounter().GetValue()
		}
	case m.Untyped != nil:
		if !math.IsNaN(m.GetUntyped().GetValue()) {
			fields[metricName] = m.GetUntyped().GetValue()
		}
	}
	return fields
}
