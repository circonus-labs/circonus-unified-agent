package suricata

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/testutil"
	"github.com/stretchr/testify/require"
)

var ex2 = `{"timestamp":"2017-03-06T07:43:39.000397+0000","event_type":"stats","stats":{"capture":{"kernel_packets":905344474,"kernel_drops":78355440,"kernel_packets_delta":2376742,"kernel_drops_delta":82049}}}`
var ex3 = `{"timestamp":"2017-03-06T07:43:39.000397+0000","event_type":"stats","stats":{"threads": { "W#05-wlp4s0": { "capture":{"kernel_packets":905344474,"kernel_drops":78355440}}}}}`

func TestSuricataLarge(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	tmpfn := filepath.Join(dir, fmt.Sprintf("t%d", rand.Int63())) //nolint:gosec // G404

	s := Suricata{
		Source:    tmpfn,
		Delimiter: ".",
		Log: testutil.Logger{
			Name: "inputs.suricata",
		},
	}
	acc := testutil.Accumulator{}
	require.NoError(t, s.Start(context.Background(), &acc))
	defer s.Stop()

	data, err := os.ReadFile("testdata/test1.json")
	require.NoError(t, err)

	c, err := net.Dial("unix", tmpfn)
	require.NoError(t, err)
	_, _ = c.Write(data)
	_, _ = c.Write([]byte("\n"))
	c.Close()

	acc.Wait(1)
}

func TestSuricata(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	tmpfn := filepath.Join(dir, fmt.Sprintf("t%d", rand.Int63())) //nolint:gosec // G404

	s := Suricata{
		Source:    tmpfn,
		Delimiter: ".",
		Log: testutil.Logger{
			Name: "inputs.suricata",
		},
	}
	acc := testutil.Accumulator{}
	require.NoError(t, s.Start(context.Background(), &acc))
	defer s.Stop()

	c, err := net.Dial("unix", tmpfn)
	require.NoError(t, err)
	_, _ = c.Write([]byte(ex2))
	_, _ = c.Write([]byte("\n"))
	c.Close()

	acc.Wait(1)

	expected := []cua.Metric{
		testutil.MustMetric(
			"suricata",
			map[string]string{
				"thread": "total",
			},
			map[string]interface{}{
				"capture.kernel_packets":       float64(905344474),
				"capture.kernel_drops":         float64(78355440),
				"capture.kernel_packets_delta": float64(2376742),
				"capture.kernel_drops_delta":   float64(82049),
			},
			time.Unix(0, 0),
		),
	}

	testutil.RequireMetricsEqual(t, expected, acc.GetCUAMetrics(), testutil.IgnoreTime())
}

func TestThreadStats(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	tmpfn := filepath.Join(dir, fmt.Sprintf("t%d", rand.Int63())) //nolint:gosec // G404

	s := Suricata{
		Source:    tmpfn,
		Delimiter: ".",
		Log: testutil.Logger{
			Name: "inputs.suricata",
		},
	}

	acc := testutil.Accumulator{}
	require.NoError(t, s.Start(context.Background(), &acc))
	defer s.Stop()

	c, err := net.Dial("unix", tmpfn)
	require.NoError(t, err)
	_, _ = c.Write([]byte(""))
	_, _ = c.Write([]byte("\n"))
	_, _ = c.Write([]byte("foobard}\n"))
	_, _ = c.Write([]byte(ex3))
	_, _ = c.Write([]byte("\n"))
	c.Close()
	acc.Wait(1)

	expected := []cua.Metric{
		testutil.MustMetric(
			"suricata",
			map[string]string{
				"thread": "W#05-wlp4s0",
			},
			map[string]interface{}{
				"capture.kernel_packets": float64(905344474),
				"capture.kernel_drops":   float64(78355440),
			},
			time.Unix(0, 0),
		),
	}

	testutil.RequireMetricsEqual(t, expected, acc.GetCUAMetrics(), testutil.IgnoreTime())
}

func TestSuricataInvalid(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	tmpfn := filepath.Join(dir, fmt.Sprintf("t%d", rand.Int63())) //nolint:gosec // G404

	s := Suricata{
		Source: tmpfn,
		Log: testutil.Logger{
			Name: "inputs.suricata",
		},
	}
	acc := testutil.Accumulator{}
	acc.SetDebug(true)

	require.NoError(t, s.Start(context.Background(), &acc))
	defer s.Stop()

	c, err := net.Dial("unix", tmpfn)
	require.NoError(t, err)
	_, _ = c.Write([]byte("sfjiowef"))
	_, _ = c.Write([]byte("\n"))
	c.Close()

	acc.WaitError(1)
}

func TestSuricataInvalidPath(t *testing.T) {
	tmpfn := fmt.Sprintf("/t%d/X", rand.Int63()) //nolint:gosec // G404
	s := Suricata{
		Source: tmpfn,
		Log: testutil.Logger{
			Name: "inputs.suricata",
		},
	}

	acc := testutil.Accumulator{}
	require.Error(t, s.Start(context.Background(), &acc))
}

func TestSuricataTooLongLine(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	tmpfn := filepath.Join(dir, fmt.Sprintf("t%d", rand.Int63())) //nolint:gosec // G404

	s := Suricata{
		Source: tmpfn,
		Log: testutil.Logger{
			Name: "inputs.suricata",
		},
	}
	acc := testutil.Accumulator{}

	require.NoError(t, s.Start(context.Background(), &acc))
	defer s.Stop()

	c, err := net.Dial("unix", tmpfn)
	require.NoError(t, err)
	_, _ = c.Write([]byte(strings.Repeat("X", 20000000)))
	_, _ = c.Write([]byte("\n"))
	c.Close()

	acc.WaitError(1)

}

func TestSuricataEmptyJSON(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	tmpfn := filepath.Join(dir, fmt.Sprintf("t%d", rand.Int63())) //nolint:gosec // G404

	s := Suricata{
		Source: tmpfn,
		Log: testutil.Logger{
			Name: "inputs.suricata",
		},
	}
	acc := testutil.Accumulator{}
	require.NoError(t, s.Start(context.Background(), &acc))
	defer s.Stop()

	c, err := net.Dial("unix", tmpfn)
	if err != nil {
		log.Println(err)

	}
	_, _ = c.Write([]byte("\n"))
	c.Close()

	acc.WaitError(1)
}

func TestSuricataDisconnectSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	tmpfn := filepath.Join(dir, fmt.Sprintf("t%d", rand.Int63())) //nolint:gosec // G404

	s := Suricata{
		Source: tmpfn,
		Log: testutil.Logger{
			Name: "inputs.suricata",
		},
	}
	acc := testutil.Accumulator{}

	require.NoError(t, s.Start(context.Background(), &acc))
	defer s.Stop()

	c, err := net.Dial("unix", tmpfn)
	require.NoError(t, err)
	_, _ = c.Write([]byte(ex2))
	_, _ = c.Write([]byte("\n"))
	c.Close()

	c, err = net.Dial("unix", tmpfn)
	require.NoError(t, err)
	_, _ = c.Write([]byte(ex3))
	_, _ = c.Write([]byte("\n"))
	c.Close()

	acc.Wait(2)
}

func TestSuricataStartStop(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	tmpfn := filepath.Join(dir, fmt.Sprintf("t%d", rand.Int63())) //nolint:gosec // G404

	s := Suricata{
		Source: tmpfn,
		Log: testutil.Logger{
			Name: "inputs.suricata",
		},
	}
	acc := testutil.Accumulator{}
	require.NoError(t, s.Start(context.Background(), &acc))
	s.Stop()
}
