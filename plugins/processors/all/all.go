package all

//nolint:golint
import (
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/clone"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/converter"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/date"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/dedup"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/defaults"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/enum"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/execd"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/filepath"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/ifname"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/override"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/parser"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/pivot"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/port_name"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/printer"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/regex"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/rename"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/reverse_dns"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/s2geo"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/starlark"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/strings"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/tag_limit"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/template"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/topk"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/processors/unpivot"
)
