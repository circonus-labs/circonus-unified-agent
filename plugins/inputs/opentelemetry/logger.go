package opentelemetry

import (
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

type otelLogger struct {
	cua.Logger
}

func (l otelLogger) Debug(msg string, kv ...interface{}) {
	format := msg + strings.Repeat(" %s=%q", len(kv)/2)
	l.Logger.Debugf(format, kv...)
}
