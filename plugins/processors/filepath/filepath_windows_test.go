//go:build windows
// +build windows

package filepath

import (
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/testutil"
)

var samplePath = "c:\\my\\test\\\\c\\..\\path\\file.log"

func TestOptions_Apply(t *testing.T) {
	tests := []testCase{
		{
			name:         "Smoke Test",
			o:            newOptions("c:\\my\\test\\"),
			inputMetrics: getSmokeTestInputMetrics(samplePath),
			expectedMetrics: []cua.Metric{
				testutil.MustMetric(
					smokeMetricName,
					map[string]string{
						"baseTag":  "file.log",
						"dirTag":   "c:\\my\\test\\path",
						"stemTag":  "file",
						"cleanTag": "c:\\my\\test\\path\\file.log",
						"relTag":   "path\\file.log",
						"slashTag": "c:/my/test//c/../path/file.log",
					},
					map[string]interface{}{
						"baseField":  "file.log",
						"dirField":   "c:\\my\\test\\path",
						"stemField":  "file",
						"cleanField": "c:\\my\\test\\path\\file.log",
						"relField":   "path\\file.log",
						"slashField": "c:/my/test//c/../path/file.log",
					},
					time.Now()),
			},
		},
	}
	runTestOptionsApply(t, tests)
}
