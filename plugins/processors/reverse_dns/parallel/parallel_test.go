package parallel_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/metric"
	"github.com/circonus-labs/circonus-unified-agent/plugins/processors/reverse_dns/parallel"
	"github.com/circonus-labs/circonus-unified-agent/testutil"
	"github.com/stretchr/testify/require"
)

func TestOrderedJobsStayOrdered(t *testing.T) {
	acc := &testutil.Accumulator{}

	p := parallel.NewOrdered(acc, jobFunc, 10000, 10)
	now := time.Now()
	for i := 0; i < 20000; i++ {
		m, err := metric.New("test",
			map[string]string{},
			map[string]interface{}{
				"val": i,
			},
			now,
		)
		require.NoError(t, err)
		now = now.Add(1)
		p.Enqueue(m)
	}
	p.Stop()

	i := 0
	require.Len(t, acc.Metrics, 20000, fmt.Sprintf("expected 20k metrics but got %d", len(acc.GetCUAMetrics())))
	for _, m := range acc.GetCUAMetrics() {
		v, ok := m.GetField("val")
		require.True(t, ok)
		require.EqualValues(t, i, v)
		i++
	}
}

func TestUnorderedJobsDontDropAnyJobs(t *testing.T) {
	acc := &testutil.Accumulator{}

	p := parallel.NewUnordered(acc, jobFunc, 10)

	now := time.Now()

	expectedTotal := 0
	for i := 0; i < 20000; i++ {
		expectedTotal += i
		m, err := metric.New("test",
			map[string]string{},
			map[string]interface{}{
				"val": i,
			},
			now,
		)
		require.NoError(t, err)
		now = now.Add(1)
		p.Enqueue(m)
	}
	p.Stop()

	actualTotal := int64(0)
	require.Len(t, acc.Metrics, 20000, fmt.Sprintf("expected 20k metrics but got %d", len(acc.GetCUAMetrics())))
	for _, m := range acc.GetCUAMetrics() {
		v, ok := m.GetField("val")
		require.True(t, ok)
		actualTotal += v.(int64)
	}
	require.EqualValues(t, expectedTotal, actualTotal)
}

func BenchmarkOrdered(b *testing.B) {
	acc := &testutil.Accumulator{}

	p := parallel.NewOrdered(acc, jobFunc, 10000, 10)

	m, _ := metric.New("test",
		map[string]string{},
		map[string]interface{}{
			"val": 1,
		},
		time.Now(),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Enqueue(m)
	}
	p.Stop()
}

func BenchmarkUnordered(b *testing.B) {
	acc := &testutil.Accumulator{}

	p := parallel.NewUnordered(acc, jobFunc, 10)

	m, _ := metric.New("test",
		map[string]string{},
		map[string]interface{}{
			"val": 1,
		},
		time.Now(),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Enqueue(m)
	}
	p.Stop()
}

func jobFunc(m cua.Metric) []cua.Metric {
	return []cua.Metric{m}
}
