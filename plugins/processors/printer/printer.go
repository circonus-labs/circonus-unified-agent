package printer

import (
	"fmt"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/processors"
	"github.com/circonus-labs/circonus-unified-agent/plugins/serializers"
	"github.com/circonus-labs/circonus-unified-agent/plugins/serializers/circonus"
)

type Printer struct {
	serializer serializers.Serializer
}

var sampleConfig = `
`

func (p *Printer) SampleConfig() string {
	return sampleConfig
}

func (p *Printer) Description() string {
	return "Print all metrics that pass through this filter."
}

func (p *Printer) Apply(in ...cua.Metric) []cua.Metric {
	for _, metric := range in {
		octets, err := p.serializer.Serialize(metric)
		if err != nil {
			continue
		}
		fmt.Printf("%s", octets)
	}
	return in
}

func init() {
	processors.Add("printer", func() cua.Processor {
		s, _ := circonus.NewSerializer(time.Millisecond)
		return &Printer{
			serializer: s,
		}
	})
}
