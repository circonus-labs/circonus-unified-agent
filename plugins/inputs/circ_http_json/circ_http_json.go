package circhttpjson

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
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

// NOTE: this input only supports direct metrics - they do NOT go through a regular output plugin

type CHJ struct {
	Log        cua.Logger
	dest       *trapmetrics.TrapMetrics
	tlsCfg     *tls.Config
	instLogger *Logshim
	TLSCN      string
	InstanceID string `json:"instance_id"`
	URL        string
	TLSCAFile  string
	Debug      bool
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

	// this is needed for retryablehttp to work...
	chj.instLogger = &Logshim{
		logh:  chj.Log,
		debug: chj.Debug,
	}

	return nil
}

func (*CHJ) Description() string {
	return "Circonus HTTP JSON retrieves HTTPTrap formatted metrics and forwards them to an HTTPTrap check"
}

func (*CHJ) SampleConfig() string {
	return `
instance_id = "" # required
url = "" # required

## optional, turn on debugging for the *metric fetch* phase of the plugin
## metric submission, to the broker, will output via regular agent debug setting.
debug = false

## Optional: tls ca cert file and common name to use
## pass if URL is https and not using a public ca
# tls_ca_cert_file = ""
# tls_cn = ""
`
}

func (chj *CHJ) Gather(ctx context.Context, _ cua.Accumulator) error {
	if chj.dest == nil {
		return fmt.Errorf("instance_id: %s -- no metric destination configured", chj.InstanceID)
	}

	data, err := chj.getURL(ctx)
	if err != nil {
		return fmt.Errorf("instance_id: %s -- fetching metrics from %s: %w", chj.InstanceID, chj.URL, err)
	}

	if err := chj.hasStreamtags(data); err != nil {
		return fmt.Errorf("instance_id: %s -- no streamtags found in metrics", chj.InstanceID)
	}

	// if err := chj.verifyJSON(data); err != nil {
	// 	return fmt.Errorf("instance_id: %s -- invalid json from %s: %w", chj.InstanceID, chj.URL, err)
	// }

	if _, err := chj.dest.FlushRawJSON(ctx, data); err != nil {
		return fmt.Errorf("instance_id: %s -- flushing metrics: %w", chj.InstanceID, err)
	}

	return nil
}

// getURL fetches the raw json from an endpoint, the JSON must adhere to circonus httptrap formatting
// can handle tagged or un-tagged json formats -- the plugin just forwards the JSON it gets to the broker
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
	retryClient.Logger = chj.instLogger
	defer retryClient.HTTPClient.CloseIdleConnections()

	resp, err := retryClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	return body, nil
}

// hasStreamtags return true if there is at least one tagged metric
func (chj *CHJ) hasStreamtags(data []byte) error {

	if len(data) == 0 {
		return fmt.Errorf("empty json")
	}

	if !bytes.Contains(data, []byte("|ST[")) {
		return fmt.Errorf("no streamtags found")
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

//
// use the simple validation in hasStreamtags, the method below is very expensive and should be used with caution
//

// type Metric struct {
// 	Value     interface{} `json:"_value"`
// 	Timestamp *uint64     `json:"_ts,omitempty"`
// 	Type      string      `json:"_type"`
// }

// type Metrics map[string]Metric

// // verifyJSON simply unmarshals a []byte into a metrics struct (defined above)
// // if it works it is considered valid -- valid JSON formatting:
// // https://docs.circonus.com/circonus/integrations/library/json-push-httptrap/#stream-tags
// func (chj *CHJ) verifyJSON(data []byte) error {
// 	if len(data) == 0 {
// 		return fmt.Errorf("empty json")
// 	}

// 	// short circuit if a tagged metric found
// 	if bytes.Contains(data, []byte("|ST[")) {
// 		return nil
// 	}

// 	var d1 bytes.Buffer
// 	if err := json.Compact(&d1, data); err != nil {
// 		return fmt.Errorf("json compact: %w", err)
// 	}

// 	if d1.Len() == 0 {
// 		return fmt.Errorf("invalid JSON (empty)")
// 	}

// 	var m Metrics
// 	if err := json.Unmarshal(d1.Bytes(), &m); err != nil {
// 		return fmt.Errorf("json unmarshal: %w", err)
// 	}

// 	if len(m) == 0 {
// 		return fmt.Errorf("invalid JSON (no metrics)")
// 	}

// 	d2, err := json.Marshal(m)
// 	if err != nil {
// 		return fmt.Errorf("json marshal: %w", err)
// 	}

// 	if d1.Len() != len(d2) {
// 		return fmt.Errorf("json invalid parse len: d1:%d != d2:%d", d1.Len(), len(d2))
// 	}

// 	return nil
// }
