package temp

import (
	"context"
	"fmt"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/system"
)

type Temperature struct {
	ps system.PS
}

func (t *Temperature) Description() string {
	return "Read metrics about temperature"
}

const sampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)
`

func (t *Temperature) SampleConfig() string {
	return sampleConfig
}

func (t *Temperature) Gather(ctx context.Context, acc cua.Accumulator) error {
	temps, err := t.ps.Temperature()
	if err != nil {
		if strings.Contains(err.Error(), "not implemented yet") {
			return fmt.Errorf("plugin is not supported on this platform: %w", err)
		}
		return fmt.Errorf("error getting temperatures info: %w", err)
	}
	for _, temp := range temps {
		tags := map[string]string{
			"sensor": temp.SensorKey,
		}
		fields := map[string]interface{}{
			"temp": temp.Temperature,
		}
		acc.AddFields("temp", fields, tags)
	}
	return nil
}

func init() {
	inputs.Add("temp", func() cua.Input {
		return &Temperature{ps: system.NewSystemPS()}
	})
}
