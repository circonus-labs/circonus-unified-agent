package health

import "github.com/circonus-labs/circonus-unified-agent/cua"

type Contains struct {
	Field string `toml:"field"`
}

func (c *Contains) Check(metrics []cua.Metric) bool {
	success := false
	for _, m := range metrics {
		ok := m.HasField(c.Field)
		if ok {
			success = true
		}
	}

	return success
}
