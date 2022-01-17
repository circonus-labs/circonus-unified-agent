package circhttpjson

import (
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

// Logshim is for retryablehttp
type Logshim struct {
	logh   cua.Logger
	prefix string
	debug  bool
}

func (l Logshim) Printf(fmt string, args ...interface{}) {
	if strings.Contains(fmt, "[DEBUG]") {
		// for retryablehttp (it only logs using Printf, and everything is DEBUG)
		if l.debug {
			l.logh.Infof(l.prefix+": "+fmt, args...)
		}
	} else {
		l.logh.Infof(l.prefix+": "+fmt, args...)
	}
}
func (l Logshim) Debugf(fmt string, args ...interface{}) {
	l.logh.Debugf(l.prefix+": "+fmt, args...)
}
func (l Logshim) Infof(fmt string, args ...interface{}) {
	l.logh.Infof(l.prefix+": "+fmt, args...)
}
func (l Logshim) Warnf(fmt string, args ...interface{}) {
	l.logh.Warnf(l.prefix+": "+fmt, args...)
}
func (l Logshim) Errorf(fmt string, args ...interface{}) {
	l.logh.Errorf(l.prefix+": "+fmt, args...)
}
