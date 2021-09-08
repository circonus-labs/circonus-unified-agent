package circhttpjson

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	circmgr "github.com/circonus-labs/circonus-unified-agent/internal/circonus"
	"github.com/circonus-labs/circonus-unified-agent/internal/release"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/go-trapmetrics"
	"github.com/hashicorp/go-retryablehttp"
)

// Collect HTTPTrap JSON payloads and forward to circonus broker
//
// 1. Use HTTP to GET metrics in valid httptrap stream tagged, structured metric format.
//  {
//    "foo|ST[env:prod,app:web]": { "_type": "n", "_value": 12 },
//    "foo|ST[env:qa,app:web]":   { "_type": "n", "_value": 0 },
//    "foo|ST[b\"fihiYXIp\":b\"PHF1dXg+\"]": { "_type": "n", "_value": 3 }
//  }
//  _type must be a valid httptrap (reconnoiter) metric type i=int,I=uint,l=int64,L=uint64,n=double,s=text,hH=histograms
//  see: https://docs.circonus.com/circonus/integrations/library/httptrap/#httptrap-json-format for more
//       information - note, metrics must use stream tag, structured formatting not arbitrary json formatting.
//
// 2. Verify metrics are formatted correctly (json.Marshal)
//
// 3. Forward to httptrap check
//
// Note: this input only supports direct metrics - they do NOT go through a regular output plugin

type Metric struct {
	Value     interface{} `json:"_value"`
	Timestamp *uint64     `json:"_ts,omitempty"`
	Type      string      `json:"_type"`
}

type Metrics map[string]Metric

type CHJ struct {
	Log        cua.Logger
	dest       *trapmetrics.TrapMetrics
	tlsCfg     *tls.Config
	InstanceID string `json:"instance_id"`
	URL        string
	TLSCAFile  string
	TLSCN      string
}

func (chj *CHJ) Init() error {

	if chj.URL == "" {
		return fmt.Errorf("invalid URL (empty)")
	}
	if chj.InstanceID == "" {
		return fmt.Errorf("invalid Instance ID (empty)")
	}

	if chj.TLSCAFile != "" {
		if err := chj.loadTLSCACert(); err != nil {
			return fmt.Errorf("loading TLSCAFile: %w", err)
		}
	}

	opts := &circmgr.MetricDestConfig{
		MetricMeta: circmgr.MetricMeta{
			PluginID:   "circ_http_json",
			InstanceID: chj.InstanceID,
		},
	}
	dest, err := circmgr.NewMetricDestination(opts, chj.Log)
	if err != nil {
		return fmt.Errorf("new metric destination: %w", err)
	}

	chj.dest = dest

	return nil
}

func (*CHJ) Description() string {
	return "Circonus HTTP JSON retrieves HTTPTrap formatted metrics and forwards them to an HTTPTrap check"
}

func (*CHJ) SampleConfig() string {
	return `
instance_id = "" # required
url = "" # required

## Optional: tls ca cert file and common name to use
## pass if URL is https and not using a public ca
# tls_ca_cert_file = ""
# tls_cn = ""
`
}

func (chj *CHJ) Gather(ctx context.Context, _ cua.Accumulator) error {
	if chj.dest != nil {
		data, err := chj.getURL(ctx)
		if err != nil {
			return err
		}

		if err := chj.verifyJSON(data); err != nil {
			return err
		}

		if _, err := chj.dest.FlushRawJSON(ctx, data); err != nil {
			return err
		}
	}
	return nil
}

// getURL fetches the raw json from an endpoint, the JSON must:
//   1. use streamtag metric names
//   2. adhere to circonus httptrap formatting
//
func (chj *CHJ) getURL(ctx context.Context) ([]byte, error) {
	var client *http.Client

	if chj.tlsCfg != nil {
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:       10 * time.Second,
					KeepAlive:     3 * time.Second,
					FallbackDelay: -1 * time.Millisecond,
				}).DialContext,
				TLSClientConfig:     chj.tlsCfg,
				TLSHandshakeTimeout: 10 * time.Second,
				DisableKeepAlives:   true,
				DisableCompression:  false,
				MaxIdleConns:        1,
				MaxIdleConnsPerHost: 0,
			},
		}
	} else {
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:       10 * time.Second,
					KeepAlive:     3 * time.Second,
					FallbackDelay: -1 * time.Millisecond,
				}).DialContext,
				DisableKeepAlives:   true,
				DisableCompression:  false,
				MaxIdleConns:        1,
				MaxIdleConnsPerHost: 0,
			},
		}
	}

	rinfo := release.GetInfo()
	req, err := retryablehttp.NewRequest("GET", chj.URL, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", rinfo.Name+"/"+rinfo.Version)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "close")

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient = client
	retryClient.Logger = chj.Log
	defer retryClient.HTTPClient.CloseIdleConnections()

	resp, err := retryClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return body, nil
}

// verifyJSON simply unmarshals a []byte into a metrics struct (defined above)
// if it works it is considered valid
func (chj *CHJ) verifyJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("invalid JSON (empty)")
	}
	var m Metrics
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}

// loadTLSCACert reads in the configured TLS CA cert file and creates
// a tls.Config to use during metric fetching from URL
func (chj *CHJ) loadTLSCACert() error {
	data, err := os.ReadFile(chj.TLSCAFile)
	if err != nil {
		return err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(data) {
		return fmt.Errorf("unable to append cert from pem %s", chj.TLSCAFile)
	}

	chj.tlsCfg = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, //nolint:gosec
		VerifyConnection: func(cs tls.ConnectionState) error {
			commonName := cs.PeerCertificates[0].Subject.CommonName
			if commonName != cs.ServerName {
				return x509.CertificateInvalidError{
					Cert:   cs.PeerCertificates[0],
					Reason: x509.NameMismatch,
					Detail: fmt.Sprintf("cn: %q, acceptable: %q", commonName, cs.ServerName),
				}
			}
			opts := x509.VerifyOptions{
				Roots:         certPool,
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := cs.PeerCertificates[0].Verify(opts)
			if err != nil {
				return fmt.Errorf("peer cert verify: %w", err)
			}
			return nil
		},
	}

	if chj.TLSCN != "" {
		chj.tlsCfg.ServerName = chj.TLSCN
	}

	return nil
}

func init() {
	inputs.Add("circ_http_json", func() cua.Input {
		return &CHJ{}
	})
}
