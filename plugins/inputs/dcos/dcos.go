package dcos

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/filter"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	jwt "github.com/dgrijalva/jwt-go/v4"
)

const (
	defaultMaxConnections  = 10
	defaultResponseTimeout = 20 * time.Second
)

var (
	nodeDimensions = []string{
		"hostname",
		"path",
		"interface",
	}
	containerDimensions = []string{
		"hostname",
		"container_id",
		"task_name",
	}
	appDimensions = []string{
		"hostname",
		"container_id",
		"task_name",
	}
)

type DCOS struct {
	appFilter       filter.Filter
	client          Client
	containerFilter filter.Filter
	nodeFilter      filter.Filter
	creds           Credentials
	tls.ClientConfig
	TokenFile                string
	ServiceAccountID         string `toml:"service_account_id"`
	ServiceAccountPrivateKey string
	ClusterURL               string `toml:"cluster_url"`
	NodeExclude              []string
	NodeInclude              []string
	AppInclude               []string
	ContainerExclude         []string
	ContainerInclude         []string
	AppExclude               []string
	MaxConnections           int
	ResponseTimeout          internal.Duration
	initialized              bool
}

func (d *DCOS) Description() string {
	return "Input plugin for DC/OS metrics"
}

var sampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)

  ## The DC/OS cluster URL.
  cluster_url = "https://dcos-ee-master-1"

  ## The ID of the service account.
  service_account_id = "circonus-unified-agent"
  ## The private key file for the service account.
  service_account_private_key = "/etc/circonus-unified-agent/circonus-unified-agent-sa-key.pem"

  ## Path containing login token.  If set, will read on every gather.
  # token_file = "/home/dcos/.dcos/token"

  ## In all filter options if both include and exclude are empty all items
  ## will be collected.  Arrays may contain glob patterns.
  ##
  ## Node IDs to collect metrics from.  If a node is excluded, no metrics will
  ## be collected for its containers or apps.
  # node_include = []
  # node_exclude = []
  ## Container IDs to collect container metrics from.
  # container_include = []
  # container_exclude = []
  ## Container IDs to collect app metrics from.
  # app_include = []
  # app_exclude = []

  ## Maximum concurrent connections to the cluster.
  # max_connections = 10
  ## Maximum time to receive a response from cluster.
  # response_timeout = "20s"

  ## Optional TLS Config
  # tls_ca = "/etc/circonus-unified-agent/ca.pem"
  # tls_cert = "/etc/circonus-unified-agent/cert.pem"
  # tls_key = "/etc/circonus-unified-agent/key.pem"
  ## If false, skip chain & host verification
  # insecure_skip_verify = true

  ## Recommended filtering to reduce series cardinality.
  # [inputs.dcos.tagdrop]
  #   path = ["/var/lib/mesos/slave/slaves/*"]
`

func (d *DCOS) SampleConfig() string {
	return sampleConfig
}

func (d *DCOS) Gather(ctx context.Context, acc cua.Accumulator) error {
	err := d.init()
	if err != nil {
		return err
	}

	token, err := d.creds.Token(ctx, d.client)
	if err != nil {
		return fmt.Errorf("token: %w", err)
	}
	d.client.SetToken(token)

	summary, err := d.client.GetSummary(ctx)
	if err != nil {
		return fmt.Errorf("summary: %w", err)
	}

	var wg sync.WaitGroup
	for _, node := range summary.Slaves {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			d.GatherNode(ctx, acc, summary.Cluster, node)
		}(node.ID)
	}
	wg.Wait()

	return nil
}

func (d *DCOS) GatherNode(ctx context.Context, acc cua.Accumulator, cluster, node string) {
	if !d.nodeFilter.Match(node) {
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m, err := d.client.GetNodeMetrics(ctx, node)
		if err != nil {
			acc.AddError(err)
			return
		}
		d.addNodeMetrics(acc, cluster, m)
	}()

	d.GatherContainers(ctx, acc, cluster, node)
	wg.Wait()
}

func (d *DCOS) GatherContainers(ctx context.Context, acc cua.Accumulator, cluster, node string) {
	containers, err := d.client.GetContainers(ctx, node)
	if err != nil {
		acc.AddError(err)
		return
	}

	var wg sync.WaitGroup
	for _, container := range containers {
		if d.containerFilter.Match(container.ID) {
			wg.Add(1)
			go func(container string) {
				defer wg.Done()
				m, err := d.client.GetContainerMetrics(ctx, node, container)
				if err != nil {
					var apiErr APIError
					if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
						return
					}
					acc.AddError(err)
					return
				}
				d.addContainerMetrics(acc, cluster, m)
			}(container.ID)
		}

		if d.appFilter.Match(container.ID) {
			wg.Add(1)
			go func(container string) {
				defer wg.Done()
				m, err := d.client.GetAppMetrics(ctx, node, container)
				if err != nil {
					var apiErr APIError
					if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
						return
					}
					acc.AddError(err)
					return
				}
				d.addAppMetrics(acc, cluster, m)
			}(container.ID)
		}
	}
	wg.Wait()
}

type point struct {
	tags   map[string]string
	labels map[string]string
	fields map[string]interface{}
}

func (d *DCOS) createPoints(m *Metrics) []*point {
	points := make(map[string]*point)
	for _, dp := range m.Datapoints {
		fieldKey := strings.ReplaceAll(dp.Name, ".", "_")

		tags := dp.Tags
		if tags == nil {
			tags = make(map[string]string)
		}

		if dp.Unit == "bytes" && !strings.HasSuffix(fieldKey, "_bytes") {
			fieldKey += "_bytes"
		}

		fieldKey = strings.TrimPrefix(fieldKey, "dcos_metrics_module_")

		tagset := make([]string, 0, len(tags))
		for k, v := range tags {
			tagset = append(tagset, k+"="+v)
		}
		sort.Strings(tagset)
		seriesParts := make([]string, 0, len(tagset))
		seriesParts = append(seriesParts, tagset...)
		seriesKey := strings.Join(seriesParts, ",")

		p, ok := points[seriesKey]
		if !ok {
			p = &point{}
			p.tags = tags
			p.labels = make(map[string]string)
			p.fields = make(map[string]interface{})
			points[seriesKey] = p
		}

		if dp.Unit == "bytes" {
			p.fields[fieldKey] = int64(dp.Value)
		} else {
			p.fields[fieldKey] = dp.Value
		}
	}

	results := make([]*point, 0, len(points))
	for _, p := range points {
		for k, v := range m.Dimensions {
			switch v := v.(type) {
			case string:
				p.tags[k] = v
			case map[string]string:
				if k == "labels" {
					for k, v := range v {
						p.labels[k] = v
					}
				}
			}
		}
		results = append(results, p)
	}
	return results
}

func (d *DCOS) addMetrics(acc cua.Accumulator, cluster, mname string, m *Metrics, tagDimensions []string) {
	tm := time.Now()

	points := d.createPoints(m)

	for _, p := range points {
		tags := make(map[string]string)
		tags["cluster"] = cluster
		for _, tagkey := range tagDimensions {
			v, ok := p.tags[tagkey]
			if ok {
				tags[tagkey] = v
			}
		}
		for k, v := range p.labels {
			tags[k] = v
		}

		acc.AddFields(mname, p.fields, tags, tm)
	}
}

func (d *DCOS) addNodeMetrics(acc cua.Accumulator, cluster string, m *Metrics) {
	d.addMetrics(acc, cluster, "dcos_node", m, nodeDimensions)
}

func (d *DCOS) addContainerMetrics(acc cua.Accumulator, cluster string, m *Metrics) {
	d.addMetrics(acc, cluster, "dcos_container", m, containerDimensions)
}

func (d *DCOS) addAppMetrics(acc cua.Accumulator, cluster string, m *Metrics) {
	d.addMetrics(acc, cluster, "dcos_app", m, appDimensions)
}

func (d *DCOS) init() error {
	if !d.initialized {
		err := d.createFilters()
		if err != nil {
			return err
		}

		if d.client == nil {
			client, err := d.createClient()
			if err != nil {
				return err
			}
			d.client = client
		}

		if d.creds == nil {
			creds, err := d.createCredentials()
			if err != nil {
				return err
			}
			d.creds = creds
		}

		d.initialized = true
	}
	return nil
}

func (d *DCOS) createClient() (Client, error) {
	tlsCfg, err := d.ClientConfig.TLSConfig()
	if err != nil {
		return nil, fmt.Errorf("TLSConfig: %w", err)
	}

	url, err := url.Parse(d.ClusterURL)
	if err != nil {
		return nil, fmt.Errorf("url parse (%s): %w", d.ClusterURL, err)
	}

	client := NewClusterClient(
		url,
		d.ResponseTimeout.Duration,
		d.MaxConnections,
		tlsCfg,
	)

	return client, nil
}

func (d *DCOS) createCredentials() (Credentials, error) {
	if d.ServiceAccountID != "" && d.ServiceAccountPrivateKey != "" {
		bs, err := os.ReadFile(d.ServiceAccountPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("readfile: %w", err)
		}

		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(bs)
		if err != nil {
			return nil, fmt.Errorf("jwt parse pem: %w", err)
		}

		creds := &ServiceAccount{
			AccountID:  d.ServiceAccountID,
			PrivateKey: privateKey,
		}
		return creds, nil
	}

	if d.TokenFile != "" {
		creds := &TokenCreds{
			Path: d.TokenFile,
		}
		return creds, nil
	}

	creds := &NullCreds{}
	return creds, nil
}

func (d *DCOS) createFilters() error {
	var err error
	d.nodeFilter, err = filter.NewIncludeExcludeFilter(d.NodeInclude, d.NodeExclude)
	if err != nil {
		return fmt.Errorf("new include/exclude filter: %w", err)
	}

	d.containerFilter, err = filter.NewIncludeExcludeFilter(d.ContainerInclude, d.ContainerExclude)
	if err != nil {
		return fmt.Errorf("new include/exclude filter: %w", err)
	}

	d.appFilter, err = filter.NewIncludeExcludeFilter(d.AppInclude, d.AppExclude)
	if err != nil {
		return fmt.Errorf("new include/exclude filter: %w", err)
	}

	return nil
}

func init() {
	inputs.Add("dcos", func() cua.Input {
		return &DCOS{
			MaxConnections: defaultMaxConnections,
			ResponseTimeout: internal.Duration{
				Duration: defaultResponseTimeout,
			},
		}
	})
}
