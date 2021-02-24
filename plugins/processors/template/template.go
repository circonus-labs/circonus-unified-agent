package template

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/processors"
)

type Processor struct {
	Tag      string     `toml:"tag"`
	Template string     `toml:"template"`
	Log      cua.Logger `toml:"-"`
	tmpl     *template.Template
}

const sampleConfig = `
  ## Tag to set with the output of the template.
  tag = "topic"

  ## Go template used to create the tag value.  In order to ease TOML
  ## escaping requirements, you may wish to use single quotes around the
  ## template string.
  template = '{{ .Tag "hostname" }}.{{ .Tag "level" }}'
`

func (r *Processor) SampleConfig() string {
	return sampleConfig
}

func (r *Processor) Description() string {
	return "Uses a Go template to create a new tag"
}

func (r *Processor) Apply(in ...cua.Metric) []cua.Metric {
	// for each metric in "in" array
	for _, metric := range in {
		var b strings.Builder
		newM := Metric{metric}

		// supply TemplateMetric and Template from configuration to Template.Execute
		err := r.tmpl.Execute(&b, &newM)
		if err != nil {
			r.Log.Errorf("failed to execute template: %v", err)
			continue
		}

		metric.AddTag(r.Tag, b.String())
	}
	return in
}

func (r *Processor) Init() error {
	// create template
	t, err := template.New("configured_template").Parse(r.Template)

	r.tmpl = t
	return fmt.Errorf("template new: %w", err)
}

func init() {
	processors.Add("template", func() cua.Processor {
		return &Processor{}
	})
}
