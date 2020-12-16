package temp

import (
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

const sampleConfig = ""

func (t *Temperature) SampleConfig() string {
	return sampleConfig
}

func (t *Temperature) Gather(acc cua.Accumulator) error {
	temps, err := t.ps.Temperature()
	if err != nil {
		if strings.Contains(err.Error(), "not implemented yet") {
			return fmt.Errorf("plugin is not supported on this platform: %v", err)
		}
		return fmt.Errorf("error getting temperatures info: %s", err)
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
