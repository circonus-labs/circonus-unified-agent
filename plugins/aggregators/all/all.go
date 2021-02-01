package all

//nolint:golint
import (
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/aggregators/basicstats"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/aggregators/final"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/aggregators/histogram"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/aggregators/merge"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/aggregators/minmax"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/aggregators/valuecounter"
)
