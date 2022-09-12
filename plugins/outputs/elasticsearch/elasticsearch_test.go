//go:build !freebsd
// +build !freebsd

package elasticsearch

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/testutil"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

const servicePort = "9200"

func launchTestContainer(t *testing.T) *testutil.Container {
	container := testutil.Container{
		Image: "elasticsearch:7.17.6",
		// Image:        "opensearchproject/opensearch:1.3.4",
		ExposedPorts: []string{servicePort},
		Env: map[string]string{
			"discovery.type": "single-node",
			// "compatibility.override_main_response_version": "true",
			// "plugins.security.disabled":                    "true",
		},
		WaitingFor: wait.ForAll(
			// wait.ForLog("] Node started"),
			wait.ForLog("] mode [basic] - valid"),
			wait.ForListeningPort(nat.Port(servicePort)),
		),
	}
	err := container.Start()
	require.NoError(t, err, "failed to start container")

	return &container
}

func TestConnectAndWriteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := launchTestContainer(t)
	defer func() {
		require.NoError(t, container.Terminate(), "terminating container failed")
	}()

	urls := []string{
		fmt.Sprintf("http://%s:%s", container.Address, container.Ports[servicePort]),
	}

	e := &Elasticsearch{
		URLs:                urls,
		IndexName:           "test-%Y.%m.%d",
		Timeout:             config.Duration(time.Second * 5),
		EnableGzip:          true,
		ManageTemplate:      true,
		TemplateName:        "circonus",
		OverwriteTemplate:   false,
		HealthCheckInterval: config.Duration(time.Second * 10),
		HealthCheckTimeout:  config.Duration(time.Second * 1),
		Log:                 testutil.Logger{},
		// Username:            "admin",
		// Password:            "admin",
		// EnableSniffer:       false,
	}

	// Verify that we can connect to Elasticsearch
	err := e.Connect()
	require.NoError(t, err)

	// Verify that we can successfully write data to Elasticsearch
	_, err = e.Write(testutil.MockMetrics())
	require.NoError(t, err)
}

func TestConnectAndWriteMetricWithNaNValueEmptyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := launchTestContainer(t)
	defer func() {
		require.NoError(t, container.Terminate(), "terminating container failed")
	}()

	urls := []string{
		fmt.Sprintf("http://%s:%s", container.Address, container.Ports[servicePort]),
	}

	e := &Elasticsearch{
		URLs:                urls,
		IndexName:           "test-%Y.%m.%d",
		Timeout:             config.Duration(time.Second * 5),
		ManageTemplate:      true,
		TemplateName:        "circonus",
		OverwriteTemplate:   false,
		HealthCheckInterval: config.Duration(time.Second * 10),
		HealthCheckTimeout:  config.Duration(time.Second * 1),
		Log:                 testutil.Logger{},
		// Username:            "admin",
		// Password:            "admin",
		// EnableSniffer:       false,
	}

	metrics := []cua.Metric{
		testutil.TestMetric(math.NaN()),
		testutil.TestMetric(math.Inf(1)),
		testutil.TestMetric(math.Inf(-1)),
	}

	// Verify that we can connect to Elasticsearch
	err := e.Connect()
	require.NoError(t, err)

	// Verify that we can fail for metric with unhandled NaN/inf/-inf values
	for _, m := range metrics {
		_, err = e.Write([]cua.Metric{m})
		require.Error(t, err, "error sending bulk request to Elasticsearch: json: unsupported value: NaN")
	}
}

func TestConnectAndWriteMetricWithNaNValueNoneIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := launchTestContainer(t)
	defer func() {
		require.NoError(t, container.Terminate(), "terminating container failed")
	}()

	urls := []string{
		fmt.Sprintf("http://%s:%s", container.Address, container.Ports[servicePort]),
	}

	e := &Elasticsearch{
		URLs:                urls,
		IndexName:           "test-%Y.%m.%d",
		Timeout:             config.Duration(time.Second * 5),
		ManageTemplate:      true,
		TemplateName:        "circonus",
		OverwriteTemplate:   false,
		HealthCheckInterval: config.Duration(time.Second * 10),
		HealthCheckTimeout:  config.Duration(time.Second * 1),
		FloatHandling:       "none",
		Log:                 testutil.Logger{},
		// Username:            "admin",
		// Password:            "admin",
		// EnableSniffer:       false,
	}

	metrics := []cua.Metric{
		testutil.TestMetric(math.NaN()),
		testutil.TestMetric(math.Inf(1)),
		testutil.TestMetric(math.Inf(-1)),
	}

	// Verify that we can connect to Elasticsearch
	err := e.Connect()
	require.NoError(t, err)

	// Verify that we can fail for metric with unhandled NaN/inf/-inf values
	for _, m := range metrics {
		_, err = e.Write([]cua.Metric{m})
		require.Error(t, err, "error sending bulk request to Elasticsearch: json: unsupported value: NaN")
	}
}

func TestConnectAndWriteMetricWithNaNValueDropIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := launchTestContainer(t)
	defer func() {
		require.NoError(t, container.Terminate(), "terminating container failed")
	}()

	urls := []string{
		fmt.Sprintf("http://%s:%s", container.Address, container.Ports[servicePort]),
	}

	e := &Elasticsearch{
		URLs:                urls,
		IndexName:           "test-%Y.%m.%d",
		Timeout:             config.Duration(time.Second * 5),
		ManageTemplate:      true,
		TemplateName:        "circonus",
		OverwriteTemplate:   false,
		HealthCheckInterval: config.Duration(time.Second * 10),
		HealthCheckTimeout:  config.Duration(time.Second * 1),
		FloatHandling:       "drop",
		Log:                 testutil.Logger{},
		// Username:            "admin",
		// Password:            "admin",
		// EnableSniffer:       false,
	}

	metrics := []cua.Metric{
		testutil.TestMetric(math.NaN()),
		testutil.TestMetric(math.Inf(1)),
		testutil.TestMetric(math.Inf(-1)),
	}

	// Verify that we can connect to Elasticsearch
	err := e.Connect()
	require.NoError(t, err)

	// Verify that we can fail for metric with unhandled NaN/inf/-inf values
	for _, m := range metrics {
		_, err = e.Write([]cua.Metric{m})
		require.NoError(t, err)
	}
}

func TestConnectAndWriteMetricWithNaNValueReplacementIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		floatHandle      string
		floatReplacement float64
		expectError      bool
	}{
		{
			floatHandle:      "none",
			floatReplacement: 0.0,
			expectError:      true,
		},
		{
			floatHandle:      "drop",
			floatReplacement: 0.0,
			expectError:      false,
		},
		{
			floatHandle:      "replace",
			floatReplacement: 0.0,
			expectError:      false,
		},
	}

	container := launchTestContainer(t)
	defer func() {
		require.NoError(t, container.Terminate(), "terminating container failed")
	}()

	urls := []string{
		fmt.Sprintf("http://%s:%s", container.Address, container.Ports[servicePort]),
	}

	for _, test := range tests {
		e := &Elasticsearch{
			URLs:                urls,
			IndexName:           "test-%Y.%m.%d",
			Timeout:             config.Duration(time.Second * 5),
			ManageTemplate:      true,
			TemplateName:        "circonus",
			OverwriteTemplate:   false,
			HealthCheckInterval: config.Duration(time.Second * 10),
			HealthCheckTimeout:  config.Duration(time.Second * 1),
			FloatHandling:       test.floatHandle,
			FloatReplacement:    test.floatReplacement,
			Log:                 testutil.Logger{},
			// Username:            "admin",
			// Password:            "admin",
			// EnableSniffer:       false,
		}

		metrics := []cua.Metric{
			testutil.TestMetric(math.NaN()),
			testutil.TestMetric(math.Inf(1)),
			testutil.TestMetric(math.Inf(-1)),
		}

		err := e.Connect()
		require.NoError(t, err)

		for _, m := range metrics {
			_, err = e.Write([]cua.Metric{m})

			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}
	}
}

func TestTemplateManagementEmptyTemplateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := launchTestContainer(t)
	defer func() {
		require.NoError(t, container.Terminate(), "terminating container failed")
	}()

	urls := []string{
		fmt.Sprintf("http://%s:%s", container.Address, container.Ports[servicePort]),
	}

	ctx := context.Background()

	e := &Elasticsearch{
		URLs:              urls,
		IndexName:         "test-%Y.%m.%d",
		Timeout:           config.Duration(time.Second * 5),
		EnableGzip:        true,
		ManageTemplate:    true,
		TemplateName:      "",
		OverwriteTemplate: true,
		Log:               testutil.Logger{},
		// Username:          "admin",
		// Password:          "admin",
		// EnableSniffer:     false,
	}

	err := e.manageTemplate(ctx)
	require.Error(t, err)
}

func TestTemplateManagementIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := launchTestContainer(t)
	defer func() {
		require.NoError(t, container.Terminate(), "terminating container failed")
	}()

	urls := []string{
		fmt.Sprintf("http://%s:%s", container.Address, container.Ports[servicePort]),
	}

	e := &Elasticsearch{
		URLs:              urls,
		IndexName:         "test-%Y.%m.%d",
		Timeout:           config.Duration(time.Second * 5),
		EnableGzip:        true,
		ManageTemplate:    true,
		TemplateName:      "circonus",
		OverwriteTemplate: true,
		Log:               testutil.Logger{},
		// Username:          "admin",
		// Password:          "admin",
		// EnableSniffer:     false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.Timeout))
	defer cancel()

	err := e.Connect()
	require.NoError(t, err)

	err = e.manageTemplate(ctx)
	require.NoError(t, err)
}

func TestTemplateInvalidIndexPatternIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := launchTestContainer(t)
	defer func() {
		require.NoError(t, container.Terminate(), "terminating container failed")
	}()

	urls := []string{
		fmt.Sprintf("http://%s:%s", container.Address, container.Ports[servicePort]),
	}

	e := &Elasticsearch{
		URLs:              urls,
		IndexName:         "{{host}}-%Y.%m.%d",
		Timeout:           config.Duration(time.Second * 5),
		EnableGzip:        true,
		ManageTemplate:    true,
		TemplateName:      "circonus",
		OverwriteTemplate: true,
		Log:               testutil.Logger{},
		// Username:          "admin",
		// Password:          "admin",
		// EnableSniffer:     false,
	}

	err := e.Connect()
	require.Error(t, err)
}

func TestGetTagKeys(t *testing.T) {
	e := &Elasticsearch{
		DefaultTagValue: "none",
		Log:             testutil.Logger{},
	}

	tests := []struct {
		IndexName         string
		ExpectedIndexName string
		ExpectedTagKeys   []string
	}{
		{
			IndexName:         "indexname",
			ExpectedIndexName: "indexname",
			ExpectedTagKeys:   []string{},
		}, {
			IndexName:         "indexname-%Y",
			ExpectedIndexName: "indexname-%Y",
			ExpectedTagKeys:   []string{},
		}, {
			IndexName:         "indexname-%Y-%m",
			ExpectedIndexName: "indexname-%Y-%m",
			ExpectedTagKeys:   []string{},
		}, {
			IndexName:         "indexname-%Y-%m-%d",
			ExpectedIndexName: "indexname-%Y-%m-%d",
			ExpectedTagKeys:   []string{},
		}, {
			IndexName:         "indexname-%Y-%m-%d-%H",
			ExpectedIndexName: "indexname-%Y-%m-%d-%H",
			ExpectedTagKeys:   []string{},
		}, {
			IndexName:         "indexname-%y-%m",
			ExpectedIndexName: "indexname-%y-%m",
			ExpectedTagKeys:   []string{},
		}, {
			IndexName:         "indexname-{{tag1}}-%y-%m",
			ExpectedIndexName: "indexname-%s-%y-%m",
			ExpectedTagKeys:   []string{"tag1"},
		}, {
			IndexName:         "indexname-{{tag1}}-{{tag2}}-%y-%m",
			ExpectedIndexName: "indexname-%s-%s-%y-%m",
			ExpectedTagKeys:   []string{"tag1", "tag2"},
		}, {
			IndexName:         "indexname-{{tag1}}-{{tag2}}-{{tag3}}-%y-%m",
			ExpectedIndexName: "indexname-%s-%s-%s-%y-%m",
			ExpectedTagKeys:   []string{"tag1", "tag2", "tag3"},
		},
	}
	for _, test := range tests {
		indexName, tagKeys := e.GetTagKeys(test.IndexName)
		if indexName != test.ExpectedIndexName {
			t.Errorf("Expected indexname %s, got %s\n", test.ExpectedIndexName, indexName)
		}
		if !reflect.DeepEqual(tagKeys, test.ExpectedTagKeys) {
			t.Errorf("Expected tagKeys %s, got %s\n", test.ExpectedTagKeys, tagKeys)
		}
	}
}

func TestGetIndexName(t *testing.T) {
	e := &Elasticsearch{
		DefaultTagValue: "none",
		Log:             testutil.Logger{},
	}

	tests := []struct {
		EventTime time.Time
		Tags      map[string]string
		IndexName string
		Expected  string
		TagKeys   []string
	}{
		{
			EventTime: time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:      map[string]string{"tag1": "value1", "tag2": "value2"},
			TagKeys:   []string{},
			IndexName: "indexname",
			Expected:  "indexname",
		},
		{
			EventTime: time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:      map[string]string{"tag1": "value1", "tag2": "value2"},
			TagKeys:   []string{},
			IndexName: "indexname-%Y",
			Expected:  "indexname-2014",
		},
		{
			EventTime: time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:      map[string]string{"tag1": "value1", "tag2": "value2"},
			TagKeys:   []string{},
			IndexName: "indexname-%Y-%m",
			Expected:  "indexname-2014-12",
		},
		{
			EventTime: time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:      map[string]string{"tag1": "value1", "tag2": "value2"},
			TagKeys:   []string{},
			IndexName: "indexname-%Y-%m-%d",
			Expected:  "indexname-2014-12-01",
		},
		{
			EventTime: time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:      map[string]string{"tag1": "value1", "tag2": "value2"},
			TagKeys:   []string{},
			IndexName: "indexname-%Y-%m-%d-%H",
			Expected:  "indexname-2014-12-01-23",
		},
		{
			EventTime: time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:      map[string]string{"tag1": "value1", "tag2": "value2"},
			TagKeys:   []string{},
			IndexName: "indexname-%y-%m",
			Expected:  "indexname-14-12",
		},
		{
			EventTime: time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:      map[string]string{"tag1": "value1", "tag2": "value2"},
			TagKeys:   []string{},
			IndexName: "indexname-%Y-%V",
			Expected:  "indexname-2014-49",
		},
		{
			EventTime: time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:      map[string]string{"tag1": "value1", "tag2": "value2"},
			TagKeys:   []string{"tag1"},
			IndexName: "indexname-%s-%y-%m",
			Expected:  "indexname-value1-14-12",
		},
		{
			EventTime: time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:      map[string]string{"tag1": "value1", "tag2": "value2"},
			TagKeys:   []string{"tag1", "tag2"},
			IndexName: "indexname-%s-%s-%y-%m",
			Expected:  "indexname-value1-value2-14-12",
		},
		{
			EventTime: time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:      map[string]string{"tag1": "value1", "tag2": "value2"},
			TagKeys:   []string{"tag1", "tag2", "tag3"},
			IndexName: "indexname-%s-%s-%s-%y-%m",
			Expected:  "indexname-value1-value2-none-14-12",
		},
	}
	for _, test := range tests {
		indexName := e.GetIndexName(test.IndexName, test.EventTime, test.TagKeys, test.Tags)
		if indexName != test.Expected {
			t.Errorf("Expected indexname %s, got %s\n", test.Expected, indexName)
		}
	}
}

func TestGetPipelineName(t *testing.T) {
	e := &Elasticsearch{
		UsePipeline:     "{{es-pipeline}}",
		DefaultPipeline: "myDefaultPipeline",
		Log:             testutil.Logger{},
	}
	e.pipelineName, e.pipelineTagKeys = e.GetTagKeys(e.UsePipeline)

	tests := []struct {
		EventTime       time.Time
		Tags            map[string]string
		Expected        string
		PipelineTagKeys []string
	}{
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1", "tag2": "value2"},
			PipelineTagKeys: []string{},
			Expected:        "myDefaultPipeline",
		},
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1", "tag2": "value2"},
			PipelineTagKeys: []string{},
			Expected:        "myDefaultPipeline",
		},
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1", "es-pipeline": "myOtherPipeline"},
			PipelineTagKeys: []string{},
			Expected:        "myOtherPipeline",
		},
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1", "es-pipeline": "pipeline2"},
			PipelineTagKeys: []string{},
			Expected:        "pipeline2",
		},
	}
	for _, test := range tests {
		pipelineName := e.getPipelineName(e.pipelineName, e.pipelineTagKeys, test.Tags)
		require.Equal(t, test.Expected, pipelineName)
	}

	// Setup testing for testing no pipeline set. All the tests in this case should return "".
	e = &Elasticsearch{
		Log: testutil.Logger{},
	}
	e.pipelineName, e.pipelineTagKeys = e.GetTagKeys(e.UsePipeline)

	for _, test := range tests {
		pipelineName := e.getPipelineName(e.pipelineName, e.pipelineTagKeys, test.Tags)
		require.Equal(t, "", pipelineName)
	}
}

func TestPipelineConfigs(t *testing.T) {
	tests := []struct {
		EventTime       time.Time
		Tags            map[string]string
		Elastic         *Elasticsearch
		Expected        string
		PipelineTagKeys []string
	}{
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1", "tag2": "value2"},
			PipelineTagKeys: []string{},
			Expected:        "",
			Elastic: &Elasticsearch{
				Log: testutil.Logger{},
			},
		},
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1", "tag2": "value2"},
			PipelineTagKeys: []string{},
			Expected:        "",
			Elastic: &Elasticsearch{
				DefaultPipeline: "myDefaultPipeline",
				Log:             testutil.Logger{},
			},
		},
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1", "es-pipeline": "myOtherPipeline"},
			PipelineTagKeys: []string{},
			Expected:        "myDefaultPipeline",
			Elastic: &Elasticsearch{
				UsePipeline: "myDefaultPipeline",
				Log:         testutil.Logger{},
			},
		},
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1", "es-pipeline": "pipeline2"},
			PipelineTagKeys: []string{},
			Expected:        "",
			Elastic: &Elasticsearch{
				DefaultPipeline: "myDefaultPipeline",
				Log:             testutil.Logger{},
			},
		},
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1", "es-pipeline": "pipeline2"},
			PipelineTagKeys: []string{},
			Expected:        "pipeline2",
			Elastic: &Elasticsearch{
				UsePipeline: "{{es-pipeline}}",
				Log:         testutil.Logger{},
			},
		},
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1", "es-pipeline": "pipeline2"},
			PipelineTagKeys: []string{},
			Expected:        "value1-pipeline2",
			Elastic: &Elasticsearch{
				UsePipeline: "{{tag1}}-{{es-pipeline}}",
				Log:         testutil.Logger{},
			},
		},
		{
			EventTime:       time.Date(2014, 12, 01, 23, 30, 00, 00, time.UTC),
			Tags:            map[string]string{"tag1": "value1"},
			PipelineTagKeys: []string{},
			Expected:        "",
			Elastic: &Elasticsearch{
				UsePipeline: "{{es-pipeline}}",
				Log:         testutil.Logger{},
			},
		},
	}

	for _, test := range tests {
		e := test.Elastic
		e.pipelineName, e.pipelineTagKeys = e.GetTagKeys(e.UsePipeline)
		pipelineName := e.getPipelineName(e.pipelineName, e.pipelineTagKeys, test.Tags)
		require.Equal(t, test.Expected, pipelineName)
	}
}

func TestRequestHeaderWhenGzipIsEnabled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_bulk":
			require.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
			require.Equal(t, "gzip", r.Header.Get("Accept-Encoding"))
			_, err := w.Write([]byte("{}"))
			require.NoError(t, err)
			return
		default:
			_, err := w.Write([]byte(`{"version": {"number": "7.8"}}`))
			require.NoError(t, err)
			return
		}
	}))
	defer ts.Close()

	urls := []string{"http://" + ts.Listener.Addr().String()}

	e := &Elasticsearch{
		URLs:           urls,
		IndexName:      "{{host}}-%Y.%m.%d",
		Timeout:        config.Duration(time.Second * 5),
		EnableGzip:     true,
		ManageTemplate: false,
		Log:            testutil.Logger{},
	}

	err := e.Connect()
	require.NoError(t, err)

	_, err = e.Write(testutil.MockMetrics())
	require.NoError(t, err)
}

func TestRequestHeaderWhenGzipIsDisabled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_bulk":
			require.NotEqual(t, "gzip", r.Header.Get("Content-Encoding"))
			_, err := w.Write([]byte("{}"))
			require.NoError(t, err)
			return
		default:
			_, err := w.Write([]byte(`{"version": {"number": "7.8"}}`))
			require.NoError(t, err)
			return
		}
	}))
	defer ts.Close()

	urls := []string{"http://" + ts.Listener.Addr().String()}

	e := &Elasticsearch{
		URLs:           urls,
		IndexName:      "{{host}}-%Y.%m.%d",
		Timeout:        config.Duration(time.Second * 5),
		EnableGzip:     false,
		ManageTemplate: false,
		Log:            testutil.Logger{},
	}

	err := e.Connect()
	require.NoError(t, err)

	_, err = e.Write(testutil.MockMetrics())
	require.NoError(t, err)
}

func TestAuthorizationHeaderWhenBearerTokenIsPresent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_bulk":
			require.Equal(t, "Bearer 0123456789abcdef", r.Header.Get("Authorization"))
			_, err := w.Write([]byte("{}"))
			require.NoError(t, err)
			return
		default:
			_, err := w.Write([]byte(`{"version": {"number": "7.8"}}`))
			require.NoError(t, err)
			return
		}
	}))
	defer ts.Close()

	urls := []string{"http://" + ts.Listener.Addr().String()}

	e := &Elasticsearch{
		URLs:            urls,
		IndexName:       "{{host}}-%Y.%m.%d",
		Timeout:         config.Duration(time.Second * 5),
		EnableGzip:      false,
		ManageTemplate:  false,
		Log:             testutil.Logger{},
		AuthBearerToken: "0123456789abcdef",
	}

	err := e.Connect()
	require.NoError(t, err)

	_, err = e.Write(testutil.MockMetrics())
	require.NoError(t, err)
}
