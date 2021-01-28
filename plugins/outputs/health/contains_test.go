package health_test

import (
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/outputs/health"
	"github.com/circonus-labs/circonus-unified-agent/testutil"
	"github.com/stretchr/testify/require"
)

func TestFieldFound(t *testing.T) {
	metrics := []cua.Metric{
		testutil.MustMetric(
			"cpu",
			map[string]string{},
			map[string]interface{}{
				"time_idle": 42.0,
			},
			time.Now()),
	}

	contains := &health.Contains{
		Field: "time_idle",
	}
	result := contains.Check(metrics)
	require.True(t, result)
}

func TestFieldNotFound(t *testing.T) {
	metrics := []cua.Metric{
		testutil.MustMetric(
			"cpu",
			map[string]string{},
			map[string]interface{}{},
			time.Now()),
	}

	contains := &health.Contains{
		Field: "time_idle",
	}
	result := contains.Check(metrics)
	require.False(t, result)
}

func TestOneMetricWithFieldIsSuccess(t *testing.T) {
	metrics := []cua.Metric{
		testutil.MustMetric(
			"cpu",
			map[string]string{},
			map[string]interface{}{},
			time.Now()),
		testutil.MustMetric(
			"cpu",
			map[string]string{},
			map[string]interface{}{
				"time_idle": 42.0,
			},
			time.Now()),
	}

	contains := &health.Contains{
		Field: "time_idle",
	}
	result := contains.Check(metrics)
	require.True(t, result)
}
