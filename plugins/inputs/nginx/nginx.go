package nginx

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type Nginx struct {
	Urls            []string
	ResponseTimeout internal.Duration
	tls.ClientConfig

	// HTTP client
	client *http.Client
}

var sampleConfig = `
  # An array of Nginx stub_status URI to gather stats.
  urls = ["http://localhost/server_status"]

  ## Optional TLS Config
  tls_ca = "/etc/circonus-unified-agent/ca.pem"
  tls_cert = "/etc/circonus-unified-agent/cert.cer"
  tls_key = "/etc/circonus-unified-agent/key.key"
  ## Use TLS but skip chain & host verification
  insecure_skip_verify = false

  # HTTP response timeout (default: 5s)
  response_timeout = "5s"
`

func (n *Nginx) SampleConfig() string {
	return sampleConfig
}

func (n *Nginx) Description() string {
	return "Read Nginx's basic status information (ngx_http_stub_status_module)"
}

func (n *Nginx) Gather(ctx context.Context, acc cua.Accumulator) error {
	var wg sync.WaitGroup

	// Create an HTTP client that is re-used for each
	// collection interval
	if n.client == nil {
		client, err := n.createHTTPClient()
		if err != nil {
			return err
		}
		n.client = client
	}

	for _, u := range n.Urls {
		addr, err := url.Parse(u)
		if err != nil {
			acc.AddError(fmt.Errorf("Unable to parse address '%s': %w", u, err))
			continue
		}

		wg.Add(1)
		go func(addr *url.URL) {
			defer wg.Done()
			acc.AddError(n.gatherURL(addr, acc))
		}(addr)
	}

	wg.Wait()
	return nil
}

func (n *Nginx) createHTTPClient() (*http.Client, error) {
	tlsCfg, err := n.ClientConfig.TLSConfig()
	if err != nil {
		return nil, fmt.Errorf("TLSConfig: %w", err)
	}

	if n.ResponseTimeout.Duration < time.Second {
		n.ResponseTimeout.Duration = time.Second * 5
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
		Timeout: n.ResponseTimeout.Duration,
	}

	return client, nil
}

func (n *Nginx) gatherURL(addr *url.URL, acc cua.Accumulator) error {
	resp, err := n.client.Get(addr.String())
	if err != nil {
		return fmt.Errorf("error making HTTP request to %s: %w", addr.String(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP status %s", addr.String(), resp.Status)
	}
	r := bufio.NewReader(resp.Body)

	// Active connections
	_, err = r.ReadString(':')
	if err != nil {
		return fmt.Errorf("read string: %w", err)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read string: %w", err)
	}
	active, err := strconv.ParseUint(strings.TrimSpace(line), 10, 64)
	if err != nil {
		return fmt.Errorf("parseuint (%s): %w", line, err)
	}

	// Server accepts handled requests
	_, err = r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read string: %w", err)
	}
	line, err = r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read string: %w", err)
	}
	data := strings.Fields(line)
	accepts, err := strconv.ParseUint(data[0], 10, 64)
	if err != nil {
		return fmt.Errorf("parseuint (%s): %w", data[0], err)
	}

	handled, err := strconv.ParseUint(data[1], 10, 64)
	if err != nil {
		return fmt.Errorf("parseuint (%s): %w", data[1], err)
	}
	requests, err := strconv.ParseUint(data[2], 10, 64)
	if err != nil {
		return fmt.Errorf("parseuint (%s): %w", data[2], err)
	}

	// Reading/Writing/Waiting
	line, err = r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read string: %w", err)
	}
	data = strings.Fields(line)
	reading, err := strconv.ParseUint(data[1], 10, 64)
	if err != nil {
		return fmt.Errorf("parseuint (%s): %w", data[1], err)
	}
	writing, err := strconv.ParseUint(data[3], 10, 64)
	if err != nil {
		return fmt.Errorf("parseuint (%s): %w", data[3], err)
	}
	waiting, err := strconv.ParseUint(data[5], 10, 64)
	if err != nil {
		return fmt.Errorf("parseuint (%s): %w", data[5], err)
	}

	tags := getTags(addr)
	fields := map[string]interface{}{
		"active":   active,
		"accepts":  accepts,
		"handled":  handled,
		"requests": requests,
		"reading":  reading,
		"writing":  writing,
		"waiting":  waiting,
	}
	acc.AddFields("nginx", fields, tags)

	return nil
}

// Get tag(s) for the nginx plugin
func getTags(addr *url.URL) map[string]string {
	h := addr.Host
	host, port, err := net.SplitHostPort(h)
	if err != nil {
		host = addr.Host
		switch addr.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			port = ""
		}
	}
	return map[string]string{"server": host, "port": port}
}

func init() {
	inputs.Add("nginx", func() cua.Input {
		return &Nginx{}
	})
}
