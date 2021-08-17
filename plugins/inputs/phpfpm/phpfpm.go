package phpfpm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/internal/globpath"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

const (
	pfPool = "pool"
	// pfProcessManager        = "process manager"
	pfStartSince         = "start since"
	pfAcceptedConn       = "accepted conn"
	pfListenQueue        = "listen queue"
	pfMaxListenQueue     = "max listen queue"
	pfListenQueueLen     = "listen queue len"
	pfIdleProcesses      = "idle processes"
	pfActiveProcesses    = "active processes"
	pfTotalProcesses     = "total processes"
	pfMaxActiveProcesses = "max active processes"
	pfMaxChildrenReached = "max children reached"
	pfSlowRequests       = "slow requests"
)

type metric map[string]int64
type poolStat map[string]metric

type phpfpm struct {
	client *http.Client
	tls.ClientConfig
	Urls    []string
	Timeout internal.Duration
}

var sampleConfig = `
  ## An array of addresses to gather stats about. Specify an ip or hostname
  ## with optional port and path
  ##
  ## Plugin can be configured in three modes (either can be used):
  ##   - http: the URL must start with http:// or https://, ie:
  ##       "http://localhost/status"
  ##       "http://192.168.130.1/status?full"
  ##
  ##   - unixsocket: path to fpm socket, ie:
  ##       "/var/run/php5-fpm.sock"
  ##      or using a custom fpm status path:
  ##       "/var/run/php5-fpm.sock:fpm-custom-status-path"
  ##
  ##   - fcgi: the URL must start with fcgi:// or cgi://, and port must be present, ie:
  ##       "fcgi://10.0.0.12:9000/status"
  ##       "cgi://10.0.10.12:9001/status"
  ##
  ## Example of multiple gathering from local socket and remote host
  ## urls = ["http://192.168.1.20/status", "/tmp/fpm.sock"]
  urls = ["http://localhost/status"]

  ## Duration allowed to complete HTTP requests.
  # timeout = "5s"

  ## Optional TLS Config
  # tls_ca = "/etc/circonus-unified-agent/ca.pem"
  # tls_cert = "/etc/circonus-unified-agent/cert.pem"
  # tls_key = "/etc/circonus-unified-agent/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = false
`

func (p *phpfpm) SampleConfig() string {
	return sampleConfig
}

func (p *phpfpm) Description() string {
	return "Read metrics of phpfpm, via HTTP status page or socket"
}

func (p *phpfpm) Init() error {
	tlsCfg, err := p.ClientConfig.TLSConfig()
	if err != nil {
		return fmt.Errorf("TLSConfig: %w", err)
	}

	p.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
		Timeout: p.Timeout.Duration,
	}
	return nil
}

// Reads stats from all configured servers accumulates stats.
// Returns one of the errors encountered while gather stats (if any).
func (p *phpfpm) Gather(ctx context.Context, acc cua.Accumulator) error {
	if len(p.Urls) == 0 {
		return p.gatherServer("http://127.0.0.1/status", acc)
	}

	var wg sync.WaitGroup

	urls, err := expandUrls(p.Urls)
	if err != nil {
		return err
	}

	for _, serv := range urls {
		wg.Add(1)
		go func(serv string) {
			defer wg.Done()
			acc.AddError(p.gatherServer(serv, acc))
		}(serv)
	}

	wg.Wait()

	return nil
}

// Request status page to get stat raw data and import it
func (p *phpfpm) gatherServer(addr string, acc cua.Accumulator) error {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return p.gatherHTTP(addr, acc)
	}

	var (
		fcgi       *conn
		socketPath string
		statusPath string
	)

	var err error
	if strings.HasPrefix(addr, "fcgi://") || strings.HasPrefix(addr, "cgi://") {
		u, err := url.Parse(addr)
		if err != nil {
			return fmt.Errorf("Unable parse server address '%s': %w", addr, err)
		}
		socketAddr := strings.Split(u.Host, ":")
		fcgiIP := socketAddr[0]
		fcgiPort, _ := strconv.Atoi(socketAddr[1])
		fcgi, err = newFcgiClient(fcgiIP, fcgiPort)
		if err != nil {
			return err
		}
		if len(u.Path) > 1 {
			statusPath = strings.Trim(u.Path, "/")
		} else {
			statusPath = "status"
		}
	} else {
		socketPath, statusPath = unixSocketPaths(addr)
		if statusPath == "" {
			statusPath = "status"
		}
		fcgi, err = newFcgiClient("unix", socketPath)
	}

	if err != nil {
		return err
	}

	return p.gatherFcgi(fcgi, statusPath, acc, addr)
}

// Gather stat using fcgi protocol
func (p *phpfpm) gatherFcgi(fcgi *conn, statusPath string, acc cua.Accumulator, addr string) error {
	fpmOutput, fpmErr, err := fcgi.Request(map[string]string{
		"SCRIPT_NAME":     "/" + statusPath,
		"SCRIPT_FILENAME": statusPath,
		"REQUEST_METHOD":  "GET",
		"CONTENT_LENGTH":  "0",
		"SERVER_PROTOCOL": "HTTP/1.0",
		"SERVER_SOFTWARE": "go / fcgiclient ",
		"REMOTE_ADDR":     "127.0.0.1",
	}, "/"+statusPath)

	if len(fpmErr) == 0 && err == nil {
		importMetric(bytes.NewReader(fpmOutput), acc, addr)
		return nil
	}
	return fmt.Errorf("Unable parse phpfpm status. Error: %v %w", string(fpmErr), err)
}

// Gather stat using http protocol
func (p *phpfpm) gatherHTTP(addr string, acc cua.Accumulator) error {
	u, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("unable parse server address '%s': %w", addr, err)
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return fmt.Errorf("unable to create new request '%s': %w", addr, err)
	}

	res, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to connect to phpfpm status page '%s': %w", addr, err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("unable to get valid stat result from '%s': %w", addr, err)
	}

	importMetric(res.Body, acc, addr)
	return nil
}

// Import stat data into system
func importMetric(r io.Reader, acc cua.Accumulator, addr string) {
	stats := make(poolStat)
	var currentPool string

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		statLine := scanner.Text()
		keyvalue := strings.Split(statLine, ":")

		if len(keyvalue) < 2 {
			continue
		}
		fieldName := strings.Trim(keyvalue[0], " ")
		// We start to gather data for a new pool here
		if fieldName == pfPool {
			currentPool = strings.Trim(keyvalue[1], " ")
			stats[currentPool] = make(metric)
			continue
		}

		// Start to parse metric for current pool
		switch fieldName {
		case pfStartSince,
			pfAcceptedConn,
			pfListenQueue,
			pfMaxListenQueue,
			pfListenQueueLen,
			pfIdleProcesses,
			pfActiveProcesses,
			pfTotalProcesses,
			pfMaxActiveProcesses,
			pfMaxChildrenReached,
			pfSlowRequests:
			fieldValue, err := strconv.ParseInt(strings.Trim(keyvalue[1], " "), 10, 64)
			if err == nil {
				stats[currentPool][fieldName] = fieldValue
			}
		}
	}

	// Finally, we push the pool metric
	for pool := range stats {
		tags := map[string]string{
			"pool": pool,
			"url":  addr,
		}
		fields := make(map[string]interface{})
		for k, v := range stats[pool] {
			fields[strings.ReplaceAll(k, " ", "_")] = v
		}
		acc.AddFields("phpfpm", fields, tags)
	}
}

func expandUrls(urls []string) ([]string, error) {
	addrs := make([]string, 0, len(urls))
	for _, url := range urls {
		if isNetworkURL(url) {
			addrs = append(addrs, url)
			continue
		}
		paths, err := globUnixSocket(url)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, paths...)
	}
	return addrs, nil
}

func globUnixSocket(url string) ([]string, error) {
	pattern, status := unixSocketPaths(url)
	glob, err := globpath.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("could not compile glob %q: %w", pattern, err)
	}
	paths := glob.Match()
	if len(paths) == 0 {
		return nil, fmt.Errorf("socket doesn't exist %q: %w", pattern, err)
	}

	addresses := make([]string, 0, len(paths))
	for _, path := range paths {
		if status != "" {
			path = path + ":" + status
		}
		addresses = append(addresses, path)
	}

	return addresses, nil
}

func unixSocketPaths(addr string) (string, string) {
	var socketPath, statusPath string

	socketAddr := strings.Split(addr, ":")
	if len(socketAddr) >= 2 {
		socketPath = socketAddr[0]
		statusPath = socketAddr[1]
	} else {
		socketPath = socketAddr[0]
		statusPath = ""
	}

	return socketPath, statusPath
}

func isNetworkURL(addr string) bool {
	return strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") || strings.HasPrefix(addr, "fcgi://") || strings.HasPrefix(addr, "cgi://")
}

func init() {
	inputs.Add("phpfpm", func() cua.Input {
		return &phpfpm{}
	})
}
