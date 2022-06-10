package json_v2_test

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/file"
	"github.com/circonus-labs/circonus-unified-agent/plugins/parsers/influx"
	"github.com/circonus-labs/circonus-unified-agent/testutil"
	"github.com/stretchr/testify/require"
)

func TestMultipleConfigs(t *testing.T) {
	// Get all directories in testdata
	folders, err := ioutil.ReadDir("testdata")
	require.NoError(t, err)
	// Make sure testdata contains data
	require.Greater(t, len(folders), 0)

	expectedErrors := []struct {
		Name  string
		Error string
	}{
		{
			Name:  "wrong_path",
			Error: "wrong",
		},
	}

	for i, f := range folders {
		if i > 1 {
			continue
		}
		t.Run(f.Name(), func(t *testing.T) {
			// Process the telegraf config file for the test
			buf, err := os.ReadFile(fmt.Sprintf("testdata/%s/circonus-unified-agent.conf", f.Name()))
			require.NoError(t, err)
			inputs.Add("file", func() cua.Input {
				return &file.File{}
			})
			cfg := config.NewConfig()
			err = cfg.LoadConfigData(buf)
			require.NoError(t, err)

			acc := testutil.Accumulator{}
			ctx := context.Background()
			for _, input := range cfg.Inputs {
				err = input.Init()
				require.NoError(t, err)
				err = input.Gather(ctx, &acc)
				// If the test has an expected error then require one was received
				var expectedError bool
				for _, e := range expectedErrors {
					if e.Name == f.Name() {
						require.Contains(t, err.Error(), e.Error)
						expectedError = true
						break
					}
				}
				if !expectedError {
					require.NoError(t, err)
				}
			}

			// Process expected metrics and compare with resulting metrics
			expectedOutputs, err := readMetricFile(fmt.Sprintf("testdata/%s/expected.out", f.Name()))
			require.NoError(t, err)
			resultingMetrics := acc.GetCUAMetrics()
			testutil.RequireMetricsEqual(t, expectedOutputs, resultingMetrics, testutil.IgnoreTime())

			// Folder with timestamp prefixed will also check for matching timestamps to make sure they are parsed correctly
			// The milliseconds weren't matching, seemed like a rounding difference between the influx parser
			// Compares each metrics times separately and ignores milliseconds
			if strings.HasPrefix(f.Name(), "timestamp") {
				require.Equal(t, len(expectedOutputs), len(resultingMetrics))
				for i, m := range resultingMetrics {
					require.Equal(t, expectedOutputs[i].Time().Truncate(time.Second), m.Time().Truncate(time.Second))
				}
			}
		})
	}
}

func readMetricFile(path string) ([]cua.Metric, error) {
	var metrics []cua.Metric
	expectedFile, err := os.Open(path)
	if err != nil {
		return metrics, err
	}
	defer expectedFile.Close()

	parser := influx.NewParser(influx.NewMetricHandler())
	scanner := bufio.NewScanner(expectedFile)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			m, err := parser.ParseLine(line)
			// The timezone needs to be UTC to match the timestamp test results
			m.SetTime(m.Time().UTC())
			if err != nil {
				return nil, fmt.Errorf("unable to parse metric in %q failed: %w", line, err)
			}
			metrics = append(metrics, m)
		}
	}
	err = expectedFile.Close()
	if err != nil {
		return metrics, err
	}

	return metrics, nil
}
