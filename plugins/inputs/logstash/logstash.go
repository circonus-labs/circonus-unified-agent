package logstash

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/internal/choice"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	jsonParser "github.com/circonus-labs/circonus-unified-agent/plugins/parsers/json"
)

const sampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)

  ## The URL of the exposed Logstash API endpoint.
  url = "http://127.0.0.1:9600"

  ## Use Logstash 5 single pipeline API, set to true when monitoring
  ## Logstash 5.
  # single_pipeline = false

  ## Enable optional collection components.  Can contain
  ## "pipelines", "process", and "jvm".
  # collect = ["pipelines", "process", "jvm"]

  ## Timeout for HTTP requests.
  # timeout = "5s"

  ## Optional HTTP Basic Auth credentials.
  # username = "username"
  # password = "pa$$word"

  ## Optional TLS Config.
  # tls_ca = "/etc/circonus-unified-agent/ca.pem"
  # tls_cert = "/etc/circonus-unified-agent/cert.pem"
  # tls_key = "/etc/circonus-unified-agent/key.pem"

  ## Use TLS but skip chain & host verification.
  # insecure_skip_verify = false

  ## Optional HTTP headers.
  # [inputs.logstash.headers]
  #   "X-Special-Header" = "Special-Value"
`

type Logstash struct {
	Headers  map[string]string `toml:"headers"`
	client   *http.Client
	URL      string `toml:"url"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	tls.ClientConfig
	Collect        []string          `toml:"collect"`
	Timeout        internal.Duration `toml:"timeout"`
	SinglePipeline bool              `toml:"single_pipeline"`
}

// NewLogstash create an instance of the plugin with default settings
func NewLogstash() *Logstash {
	return &Logstash{
		URL:            "http://127.0.0.1:9600",
		SinglePipeline: false,
		Collect:        []string{"pipelines", "process", "jvm"},
		Headers:        make(map[string]string),
		Timeout:        internal.Duration{Duration: time.Second * 5},
	}
}

// Description returns short info about plugin
func (logstash *Logstash) Description() string {
	return "Read metrics exposed by Logstash"
}

// SampleConfig returns details how to configure plugin
func (logstash *Logstash) SampleConfig() string {
	return sampleConfig
}

type ProcessStats struct {
	ID      string      `json:"id"`
	Process interface{} `json:"process"`
	Name    string      `json:"name"`
	Host    string      `json:"host"`
	Version string      `json:"version"`
}

type JVMStats struct {
	ID      string      `json:"id"`
	JVM     interface{} `json:"jvm"`
	Name    string      `json:"name"`
	Host    string      `json:"host"`
	Version string      `json:"version"`
}

type PipelinesStats struct {
	ID        string              `json:"id"`
	Pipelines map[string]Pipeline `json:"pipelines"`
	Name      string              `json:"name"`
	Host      string              `json:"host"`
	Version   string              `json:"version"`
}

type PipelineStats struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Host     string   `json:"host"`
	Version  string   `json:"version"`
	Pipeline Pipeline `json:"pipeline"`
}

type Pipeline struct {
	Queue   PipelineQueue   `json:"queue"`
	Events  interface{}     `json:"events"`
	Reloads interface{}     `json:"reloads"`
	Plugins PipelinePlugins `json:"plugins"`
}

type Plugin struct {
	ID     string      `json:"id"`
	Events interface{} `json:"events"`
	Name   string      `json:"name"`
}

type PipelinePlugins struct {
	Inputs  []Plugin `json:"inputs"`
	Filters []Plugin `json:"filters"`
	Outputs []Plugin `json:"outputs"`
}

type PipelineQueue struct {
	Capacity interface{} `json:"capacity"`
	Data     interface{} `json:"data"`
	Type     string      `json:"type"`
	Events   float64     `json:"events"`
}

const jvmStats = "/_node/stats/jvm"
const processStats = "/_node/stats/process"
const pipelinesStats = "/_node/stats/pipelines"
const pipelineStats = "/_node/stats/pipeline"

func (logstash *Logstash) Init() error {
	err := choice.CheckSlice(logstash.Collect, []string{"pipelines", "process", "jvm"})
	if err != nil {
		return fmt.Errorf(`cannot verify "collect" setting: %w`, err)
	}
	return nil
}

// createHTTPClient create a clients to access API
func (logstash *Logstash) createHTTPClient() (*http.Client, error) {
	tlsConfig, err := logstash.ClientConfig.TLSConfig()
	if err != nil {
		return nil, fmt.Errorf("TLSConfig: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: logstash.Timeout.Duration,
	}

	return client, nil
}

// gatherJSONData query the data source and parse the response JSON
func (logstash *Logstash) gatherJSONData(url string, value interface{}) error {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("http new req (%s): %w", url, err)
	}

	if (logstash.Username != "") || (logstash.Password != "") {
		request.SetBasicAuth(logstash.Username, logstash.Password)
	}

	for header, value := range logstash.Headers {
		if strings.ToLower(header) == "host" {
			request.Host = value
		} else {
			request.Header.Add(header, value)
		}
	}

	response, err := logstash.client.Do(request)
	if err != nil {
		return fmt.Errorf("client do: %w", err)
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		// ignore the err here; LimitReader returns io.EOF and we're not interested in read errors.
		body, _ := io.ReadAll(io.LimitReader(response.Body, 200))
		return fmt.Errorf("%s returned HTTP status %s: %q", url, response.Status, body)
	}

	if err := json.NewDecoder(response.Body).Decode(value); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}

	return nil
}

// gatherJVMStats gather the JVM metrics and add results to the accumulator
func (logstash *Logstash) gatherJVMStats(url string, accumulator cua.Accumulator) error {
	jvmStats := &JVMStats{}

	err := logstash.gatherJSONData(url, jvmStats)
	if err != nil {
		return err
	}

	tags := map[string]string{
		"node_id":      jvmStats.ID,
		"node_name":    jvmStats.Name,
		"node_version": jvmStats.Version,
		"source":       jvmStats.Host,
	}

	flattener := jsonParser.Flattener{}
	err = flattener.FlattenJSON("", jvmStats.JVM)
	if err != nil {
		return fmt.Errorf("flatten json: %w", err)
	}
	accumulator.AddFields("logstash_jvm", flattener.Fields, tags)

	return nil
}

// gatherJVMStats gather the Process metrics and add results to the accumulator
func (logstash *Logstash) gatherProcessStats(url string, accumulator cua.Accumulator) error {
	processStats := &ProcessStats{}

	err := logstash.gatherJSONData(url, processStats)
	if err != nil {
		return err
	}

	tags := map[string]string{
		"node_id":      processStats.ID,
		"node_name":    processStats.Name,
		"node_version": processStats.Version,
		"source":       processStats.Host,
	}

	flattener := jsonParser.Flattener{}
	err = flattener.FlattenJSON("", processStats.Process)
	if err != nil {
		return fmt.Errorf("flatten json: %w", err)
	}
	accumulator.AddFields("logstash_process", flattener.Fields, tags)

	return nil
}

// gatherPluginsStats go through a list of plugins and add their metrics to the accumulator
func (logstash *Logstash) gatherPluginsStats(
	plugins []Plugin,
	pluginType string,
	tags map[string]string,
	accumulator cua.Accumulator) error {

	for _, plugin := range plugins {
		pluginTags := map[string]string{
			"plugin_name": plugin.Name,
			"plugin_id":   plugin.ID,
			"plugin_type": pluginType,
		}
		for tag, value := range tags {
			pluginTags[tag] = value
		}
		flattener := jsonParser.Flattener{}
		err := flattener.FlattenJSON("", plugin.Events)
		if err != nil {
			return fmt.Errorf("flatten json: %w", err)
		}
		accumulator.AddFields("logstash_plugins", flattener.Fields, pluginTags)
	}

	return nil
}

func (logstash *Logstash) gatherQueueStats(
	queue *PipelineQueue,
	tags map[string]string,
	accumulator cua.Accumulator) error {

	var err error
	queueTags := map[string]string{
		"queue_type": queue.Type,
	}
	for tag, value := range tags {
		queueTags[tag] = value
	}

	queueFields := map[string]interface{}{
		"events": queue.Events,
	}

	if queue.Type != "memory" {
		flattener := jsonParser.Flattener{}
		err = flattener.FlattenJSON("", queue.Capacity)
		if err != nil {
			return fmt.Errorf("flatten json: %w", err)
		}
		err = flattener.FlattenJSON("", queue.Data)
		if err != nil {
			return fmt.Errorf("flatten json: %w", err)
		}
		for field, value := range flattener.Fields {
			queueFields[field] = value
		}
	}

	accumulator.AddFields("logstash_queue", queueFields, queueTags)

	return nil
}

// gatherJVMStats gather the Pipeline metrics and add results to the accumulator (for Logstash < 6)
func (logstash *Logstash) gatherPipelineStats(url string, accumulator cua.Accumulator) error {
	pipelineStats := &PipelineStats{}

	err := logstash.gatherJSONData(url, pipelineStats)
	if err != nil {
		return err
	}

	tags := map[string]string{
		"node_id":      pipelineStats.ID,
		"node_name":    pipelineStats.Name,
		"node_version": pipelineStats.Version,
		"source":       pipelineStats.Host,
	}

	flattener := jsonParser.Flattener{}
	err = flattener.FlattenJSON("", pipelineStats.Pipeline.Events)
	if err != nil {
		return fmt.Errorf("flatten json: %w", err)
	}
	accumulator.AddFields("logstash_events", flattener.Fields, tags)

	err = logstash.gatherPluginsStats(pipelineStats.Pipeline.Plugins.Inputs, "input", tags, accumulator)
	if err != nil {
		return err
	}
	err = logstash.gatherPluginsStats(pipelineStats.Pipeline.Plugins.Filters, "filter", tags, accumulator)
	if err != nil {
		return err
	}
	err = logstash.gatherPluginsStats(pipelineStats.Pipeline.Plugins.Outputs, "output", tags, accumulator)
	if err != nil {
		return err
	}

	err = logstash.gatherQueueStats(&pipelineStats.Pipeline.Queue, tags, accumulator)
	if err != nil {
		return err
	}

	return nil
}

// gatherJVMStats gather the Pipelines metrics and add results to the accumulator (for Logstash >= 6)
func (logstash *Logstash) gatherPipelinesStats(url string, accumulator cua.Accumulator) error {
	pipelinesStats := &PipelinesStats{}

	err := logstash.gatherJSONData(url, pipelinesStats)
	if err != nil {
		return err
	}

	for pipelineName, pipeline := range pipelinesStats.Pipelines {
		pipeline := pipeline
		tags := map[string]string{
			"node_id":      pipelinesStats.ID,
			"node_name":    pipelinesStats.Name,
			"node_version": pipelinesStats.Version,
			"pipeline":     pipelineName,
			"source":       pipelinesStats.Host,
		}

		flattener := jsonParser.Flattener{}
		err := flattener.FlattenJSON("", pipeline.Events)
		if err != nil {
			return fmt.Errorf("flatten json: %w", err)
		}
		accumulator.AddFields("logstash_events", flattener.Fields, tags)

		err = logstash.gatherPluginsStats(pipeline.Plugins.Inputs, "input", tags, accumulator)
		if err != nil {
			return err
		}
		err = logstash.gatherPluginsStats(pipeline.Plugins.Filters, "filter", tags, accumulator)
		if err != nil {
			return err
		}
		err = logstash.gatherPluginsStats(pipeline.Plugins.Outputs, "output", tags, accumulator)
		if err != nil {
			return err
		}

		err = logstash.gatherQueueStats(&pipeline.Queue, tags, accumulator)
		if err != nil {
			return err
		}
	}

	return nil
}

// Gather ask this plugin to start gathering metrics
func (logstash *Logstash) Gather(ctx context.Context, accumulator cua.Accumulator) error {
	if logstash.client == nil {
		client, err := logstash.createHTTPClient()

		if err != nil {
			return err
		}
		logstash.client = client
	}

	if choice.Contains("jvm", logstash.Collect) {
		jvmURL, err := url.Parse(logstash.URL + jvmStats)
		if err != nil {
			return fmt.Errorf("url parse (%s): %w", logstash.URL+jvmStats, err)
		}
		if err := logstash.gatherJVMStats(jvmURL.String(), accumulator); err != nil {
			return err
		}
	}

	if choice.Contains("process", logstash.Collect) {
		processURL, err := url.Parse(logstash.URL + processStats)
		if err != nil {
			return fmt.Errorf("url parse (%s): %w", logstash.URL+processStats, err)
		}
		if err := logstash.gatherProcessStats(processURL.String(), accumulator); err != nil {
			return err
		}
	}

	if choice.Contains("pipelines", logstash.Collect) {
		if logstash.SinglePipeline {
			pipelineURL, err := url.Parse(logstash.URL + pipelineStats)
			if err != nil {
				return fmt.Errorf("url parse (%s): %w", logstash.URL+pipelineStats, err)
			}
			if err := logstash.gatherPipelineStats(pipelineURL.String(), accumulator); err != nil {
				return err
			}
		} else {
			pipelinesURL, err := url.Parse(logstash.URL + pipelinesStats)
			if err != nil {
				return fmt.Errorf("url parse (%s): %w", logstash.URL+pipelinesStats, err)
			}
			if err := logstash.gatherPipelinesStats(pipelinesURL.String(), accumulator); err != nil {
				return err
			}
		}
	}

	return nil
}

// init registers this plugin instance
func init() {
	inputs.Add("logstash", func() cua.Input {
		return NewLogstash()
	})
}
