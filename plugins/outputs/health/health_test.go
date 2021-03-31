package health_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/outputs/health"
	"github.com/circonus-labs/circonus-unified-agent/testutil"
	"github.com/stretchr/testify/require"
)

var pki = testutil.NewPKI("../../../testutil/pki")

func TestHealth(t *testing.T) {
	type Options struct {
		Compares []*health.Compares `toml:"compares"`
		Contains []*health.Contains `toml:"contains"`
	}

	now := time.Now()
	tests := []struct {
		name         string
		options      Options
		metrics      []cua.Metric
		expectedCode int
	}{
		{
			name:         "healthy on startup",
			expectedCode: 200,
		},
		{
			name: "check passes",
			options: Options{
				Compares: []*health.Compares{
					{
						Field: "time_idle",
						GT:    func() *float64 { v := 0.0; return &v }(),
					},
				},
			},
			metrics: []cua.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{
						"time_idle": 42,
					},
					now),
			},
			expectedCode: 200,
		},
		{
			name: "check fails",
			options: Options{
				Compares: []*health.Compares{
					{
						Field: "time_idle",
						LT:    func() *float64 { v := 0.0; return &v }(),
					},
				},
			},
			metrics: []cua.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{
						"time_idle": 42,
					},
					now),
			},
			expectedCode: 503,
		},
		{
			name: "mixed check fails",
			options: Options{
				Compares: []*health.Compares{
					{
						Field: "time_idle",
						LT:    func() *float64 { v := 0.0; return &v }(),
					},
				},
				Contains: []*health.Contains{
					{
						Field: "foo",
					},
				},
			},
			metrics: []cua.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{
						"time_idle": 42,
					},
					now),
			},
			expectedCode: 503,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			output := health.NewHealth()
			output.ServiceAddress = "tcp://127.0.0.1:0"
			output.Compares = tt.options.Compares
			output.Contains = tt.options.Contains

			err := output.Init()
			require.NoError(t, err)

			err = output.Connect()
			require.NoError(t, err)

			_, err = output.Write(tt.metrics)
			require.NoError(t, err)

			resp, err := http.Get(output.Origin())
			require.NoError(t, err)
			require.Equal(t, tt.expectedCode, resp.StatusCode)

			_, err = io.ReadAll(resp.Body)
			require.NoError(t, err)

			err = output.Close()
			require.NoError(t, err)
		})
	}
}

func TestInitServiceAddress(t *testing.T) {
	tests := []struct {
		name   string
		plugin *health.Health
		err    bool
		origin string
	}{
		{
			name: "port without scheme is not allowed",
			plugin: &health.Health{
				ServiceAddress: ":8080",
			},
			err: true,
		},
		{
			name: "path without scheme is not allowed",
			plugin: &health.Health{
				ServiceAddress: "/tmp/cua",
			},
			err: true,
		},
		{
			name: "tcp with port maps to http",
			plugin: &health.Health{
				ServiceAddress: "tcp://:8080",
			},
		},
		{
			name: "tcp with tlsconf maps to https",
			plugin: &health.Health{
				ServiceAddress: "tcp://:8080",
				ServerConfig:   *pki.TLSServerConfig(),
			},
		},
		{
			name: "tcp4 is allowed",
			plugin: &health.Health{
				ServiceAddress: "tcp4://:8080",
			},
		},
		{
			name: "tcp6 is allowed",
			plugin: &health.Health{
				ServiceAddress: "tcp6://:8080",
			},
		},
		{
			name: "http scheme",
			plugin: &health.Health{
				ServiceAddress: "http://:8080",
			},
		},
		{
			name: "https scheme",
			plugin: &health.Health{
				ServiceAddress: "https://:8080",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			output := health.NewHealth()
			output.ServiceAddress = tt.plugin.ServiceAddress

			err := output.Init()
			if tt.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
