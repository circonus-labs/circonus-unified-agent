package testutil

import (
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/metric"
	"github.com/google/go-cmp/cmp"
)

func TestRequireMetricEqual(t *testing.T) {
	tests := []struct {
		name string
		got  cua.Metric
		want cua.Metric
	}{
		{
			name: "equal metrics should be equal",
			got: func() cua.Metric {
				m, _ := metric.New(
					"test",
					map[string]string{
						"t1": "v1",
						"t2": "v2",
					},
					map[string]interface{}{
						"f1": 1,
						"f2": 3.14,
						"f3": "v3",
					},
					time.Unix(0, 0),
				)
				return m
			}(),
			want: func() cua.Metric {
				m, _ := metric.New(
					"test",
					map[string]string{
						"t1": "v1",
						"t2": "v2",
					},
					map[string]interface{}{
						"f1": int64(1),
						"f2": 3.14,
						"f3": "v3",
					},
					time.Unix(0, 0),
				)
				return m
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			RequireMetricEqual(t, tt.want, tt.got)
		})
	}
}

func TestRequireMetricsEqual(t *testing.T) {
	tests := []struct {
		name string
		got  []cua.Metric
		want []cua.Metric
		opts []cmp.Option
	}{
		{
			name: "sort metrics option sorts by name",
			got: []cua.Metric{
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
			},
			want: []cua.Metric{
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
			},
			opts: []cmp.Option{SortMetrics()},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			RequireMetricsEqual(t, tt.want, tt.got, tt.opts...)
		})
	}
}
