package temp

import (
	"context"
	"testing"

	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/system"
	"github.com/circonus-labs/circonus-unified-agent/testutil"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/stretchr/testify/require"
)

func TestTemperature(t *testing.T) {
	var mps system.MockPS
	var err error
	defer mps.AssertExpectations(t)
	var acc testutil.Accumulator

	ts := host.TemperatureStat{
		SensorKey:   "coretemp_sensor1_crit",
		Temperature: 60.5,
	}

	mps.On("Temperature").Return([]host.TemperatureStat{ts}, nil)

	err = (&Temperature{ps: &mps}).Gather(context.Background(), &acc)
	require.NoError(t, err)

	expectedFields := map[string]interface{}{
		"temp": float64(60.5),
	}

	expectedTags := map[string]string{
		"sensor": "coretemp_sensor1_crit",
	}
	acc.AssertContainsTaggedFields(t, "temp", expectedFields, expectedTags)

}
