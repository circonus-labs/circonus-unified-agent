package tengine

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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

type Tengine struct {
	Urls            []string
	ResponseTimeout internal.Duration
	tls.ClientConfig

	client *http.Client
}

var sampleConfig = `
  # An array of Tengine reqstat module URI to gather stats.
  urls = ["http://127.0.0.1/us"]

  # HTTP response timeout (default: 5s)
  # response_timeout = "5s"

  ## Optional TLS Config
  # tls_ca = "/etc/circonus-unified-agent/ca.pem"
  # tls_cert = "/etc/circonus-unified-agent/cert.cer"
  # tls_key = "/etc/circonus-unified-agent/key.key"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = false
`

func (n *Tengine) SampleConfig() string {
	return sampleConfig
}

func (n *Tengine) Description() string {
	return "Read Tengine's basic status information (ngx_http_reqstat_module)"
}

func (n *Tengine) Gather(acc cua.Accumulator) error {
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

func (n *Tengine) createHTTPClient() (*http.Client, error) {
	tlsCfg, err := n.ClientConfig.TLSConfig()
	if err != nil {
		return nil, err
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

type Status struct {
	host                  string
	bytesIn               uint64
	bytesOut              uint64
	connTotal             uint64
	reqTotal              uint64
	http2xx               uint64
	http3xx               uint64
	http4xx               uint64
	http5xx               uint64
	httpOtherStatus       uint64
	rt                    uint64
	upsReq                uint64
	upsRt                 uint64
	upsTries              uint64
	http200               uint64
	http206               uint64
	http302               uint64
	http304               uint64
	http403               uint64
	http404               uint64
	http416               uint64
	http499               uint64
	http500               uint64
	http502               uint64
	http503               uint64
	http504               uint64
	http508               uint64
	httpOtherDetailStatus uint64
	httpUps4xx            uint64
	httpUps5xx            uint64
}

func (n *Tengine) gatherURL(addr *url.URL, acc cua.Accumulator) error {
	var tenginestatus Status
	resp, err := n.client.Get(addr.String())
	if err != nil {
		return fmt.Errorf("error making HTTP request to %s: %w", addr.String(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP status %s", addr.String(), resp.Status)
	}
	r := bufio.NewReader(resp.Body)

	for {
		line, err := r.ReadString('\n')

		if err != nil || errors.Is(err, io.EOF) {
			break
		}
		lineSplit := strings.Split(strings.TrimSpace(line), ",")
		if len(lineSplit) != 30 {
			continue
		}
		tenginestatus.host = lineSplit[0]
		if err != nil {
			return err
		}
		tenginestatus.bytesIn, err = strconv.ParseUint(lineSplit[1], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.bytesOut, err = strconv.ParseUint(lineSplit[2], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.connTotal, err = strconv.ParseUint(lineSplit[3], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.reqTotal, err = strconv.ParseUint(lineSplit[4], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http2xx, err = strconv.ParseUint(lineSplit[5], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http3xx, err = strconv.ParseUint(lineSplit[6], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http4xx, err = strconv.ParseUint(lineSplit[7], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http5xx, err = strconv.ParseUint(lineSplit[8], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.httpOtherStatus, err = strconv.ParseUint(lineSplit[9], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.rt, err = strconv.ParseUint(lineSplit[10], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.upsReq, err = strconv.ParseUint(lineSplit[11], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.upsRt, err = strconv.ParseUint(lineSplit[12], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.upsTries, err = strconv.ParseUint(lineSplit[13], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http200, err = strconv.ParseUint(lineSplit[14], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http206, err = strconv.ParseUint(lineSplit[15], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http302, err = strconv.ParseUint(lineSplit[16], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http304, err = strconv.ParseUint(lineSplit[17], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http403, err = strconv.ParseUint(lineSplit[18], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http404, err = strconv.ParseUint(lineSplit[19], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http416, err = strconv.ParseUint(lineSplit[20], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http499, err = strconv.ParseUint(lineSplit[21], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http500, err = strconv.ParseUint(lineSplit[22], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http502, err = strconv.ParseUint(lineSplit[23], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http503, err = strconv.ParseUint(lineSplit[24], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http504, err = strconv.ParseUint(lineSplit[25], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.http508, err = strconv.ParseUint(lineSplit[26], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.httpOtherDetailStatus, err = strconv.ParseUint(lineSplit[27], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.httpUps4xx, err = strconv.ParseUint(lineSplit[28], 10, 64)
		if err != nil {
			return err
		}
		tenginestatus.httpUps5xx, err = strconv.ParseUint(lineSplit[29], 10, 64)
		if err != nil {
			return err
		}
		tags := getTags(addr, tenginestatus.host)
		fields := map[string]interface{}{
			"bytes_in":                 tenginestatus.bytesIn,
			"bytes_out":                tenginestatus.bytesOut,
			"conn_total":               tenginestatus.connTotal,
			"req_total":                tenginestatus.reqTotal,
			"http_2xx":                 tenginestatus.http2xx,
			"http_3xx":                 tenginestatus.http3xx,
			"http_4xx":                 tenginestatus.http4xx,
			"http_5xx":                 tenginestatus.http5xx,
			"http_other_status":        tenginestatus.httpOtherStatus,
			"rt":                       tenginestatus.rt,
			"ups_req":                  tenginestatus.upsReq,
			"ups_rt":                   tenginestatus.upsRt,
			"ups_tries":                tenginestatus.upsTries,
			"http_200":                 tenginestatus.http200,
			"http_206":                 tenginestatus.http206,
			"http_302":                 tenginestatus.http302,
			"http_304":                 tenginestatus.http304,
			"http_403":                 tenginestatus.http403,
			"http_404":                 tenginestatus.http404,
			"http_416":                 tenginestatus.http416,
			"http_499":                 tenginestatus.http499,
			"http_500":                 tenginestatus.http500,
			"http_502":                 tenginestatus.http502,
			"http_503":                 tenginestatus.http503,
			"http_504":                 tenginestatus.http504,
			"http_508":                 tenginestatus.http508,
			"http_other_detail_status": tenginestatus.httpOtherDetailStatus,
			"http_ups_4xx":             tenginestatus.httpUps4xx,
			"http_ups_5xx":             tenginestatus.httpUps5xx,
		}
		acc.AddFields("tengine", fields, tags)
	}

	return nil
}

// Get tag(s) for the tengine plugin
func getTags(addr *url.URL, serverName string) map[string]string {
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
	return map[string]string{"server": host, "port": port, "server_name": serverName}
}

func init() {
	inputs.Add("tengine", func() cua.Input {
		return &Tengine{}
	})
}
