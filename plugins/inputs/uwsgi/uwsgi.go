// Package uwsgi implements a plugin for collecting uwsgi stats from
// the uwsgi stats server.
package uwsgi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

// Uwsgi server struct
type Uwsgi struct {
	Servers []string          `toml:"servers"`
	Timeout internal.Duration `toml:"timeout"`

	client *http.Client
}

// Description returns the plugin description
func (u *Uwsgi) Description() string {
	return "Read uWSGI metrics."
}

// SampleConfig returns the sample configuration
func (u *Uwsgi) SampleConfig() string {
	return `
  instance_id = "" # unique instance identifier (REQUIRED)

  ## List with urls of uWSGI Stats servers. URL must match pattern:
  ## scheme://address[:port]
  ##
  ## For example:
  ## servers = ["tcp://localhost:5050", "http://localhost:1717", "unix:///tmp/statsock"]
  servers = ["tcp://127.0.0.1:1717"]

  ## General connection timeout
  # timeout = "5s"
`
}

// Gather collect data from uWSGI Server
func (u *Uwsgi) Gather(ctx context.Context, acc cua.Accumulator) error {
	if u.client == nil {
		u.client = &http.Client{
			Timeout: u.Timeout.Duration,
		}
	}
	wg := &sync.WaitGroup{}

	for _, s := range u.Servers {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			n, err := url.Parse(s)
			if err != nil {
				acc.AddError(fmt.Errorf("could not parse uWSGI Stats Server url '%s': %w", s, err))
				return
			}

			if err := u.gatherServer(acc, n); err != nil {
				acc.AddError(err)
				return
			}
		}(s)
	}

	wg.Wait()

	return nil
}

func (u *Uwsgi) gatherServer(acc cua.Accumulator, surl *url.URL) error {
	var err error
	var r io.ReadCloser
	var s StatsServer

	switch surl.Scheme {
	case "tcp":
		r, err = net.DialTimeout(surl.Scheme, surl.Host, u.Timeout.Duration)
		if err != nil {
			return fmt.Errorf("dial (%s): %w", surl.Host, err)
		}
		s.source = surl.Host
	case "unix":
		r, err = net.DialTimeout(surl.Scheme, surl.Path, u.Timeout.Duration)
		if err != nil {
			return fmt.Errorf("dial (%s): %w", surl.Path, err)
		}
		s.source, err = os.Hostname()
		if err != nil {
			s.source = ""
		}
	case "http":
		resp, err := u.client.Get(surl.String())
		if err != nil {
			return fmt.Errorf("http get (%s): %w", surl.String(), err)
		}
		r = resp.Body
		s.source = surl.Host
	default:
		return fmt.Errorf("'%s' is not a supported scheme", surl.Scheme)
	}

	defer r.Close()

	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return fmt.Errorf("failed to decode json payload from '%s': %w", surl.String(), err)
	}

	u.gatherStatServer(acc, &s)

	return nil
}

func (u *Uwsgi) gatherStatServer(acc cua.Accumulator, s *StatsServer) {
	fields := map[string]interface{}{
		"listen_queue":        s.ListenQueue,
		"listen_queue_errors": s.ListenQueueErrors,
		"signal_queue":        s.SignalQueue,
		"load":                s.Load,
		"pid":                 s.PID,
	}

	tags := map[string]string{
		"source":  s.source,
		"uid":     strconv.Itoa(s.UID),
		"gid":     strconv.Itoa(s.GID),
		"version": s.Version,
	}
	acc.AddFields("uwsgi_overview", fields, tags)

	u.gatherWorkers(acc, s)
	u.gatherApps(acc, s)
	u.gatherCores(acc, s)
}

func (u *Uwsgi) gatherWorkers(acc cua.Accumulator, s *StatsServer) {
	for _, w := range s.Workers {
		fields := map[string]interface{}{
			"requests":       w.Requests,
			"accepting":      w.Accepting,
			"delta_request":  w.DeltaRequests,
			"exceptions":     w.Exceptions,
			"harakiri_count": w.HarakiriCount,
			"pid":            w.PID,
			"signals":        w.Signals,
			"signal_queue":   w.SignalQueue,
			"status":         w.Status,
			"rss":            w.Rss,
			"vsz":            w.Vsz,
			"running_time":   w.RunningTime,
			"last_spawn":     w.LastSpawn,
			"respawn_count":  w.RespawnCount,
			"tx":             w.Tx,
			"avg_rt":         w.AvgRt,
		}
		tags := map[string]string{
			"worker_id": strconv.Itoa(w.WorkerID),
			"source":    s.source,
		}

		acc.AddFields("uwsgi_workers", fields, tags)
	}
}

func (u *Uwsgi) gatherApps(acc cua.Accumulator, s *StatsServer) {
	for _, w := range s.Workers {
		for _, a := range w.Apps {
			fields := map[string]interface{}{
				"modifier1":    a.Modifier1,
				"requests":     a.Requests,
				"startup_time": a.StartupTime,
				"exceptions":   a.Exceptions,
			}
			tags := map[string]string{
				"app_id":    strconv.Itoa(a.AppID),
				"worker_id": strconv.Itoa(w.WorkerID),
				"source":    s.source,
			}
			acc.AddFields("uwsgi_apps", fields, tags)
		}
	}
}

func (u *Uwsgi) gatherCores(acc cua.Accumulator, s *StatsServer) {
	for _, w := range s.Workers {
		for _, c := range w.Cores {
			fields := map[string]interface{}{
				"requests":           c.Requests,
				"static_requests":    c.StaticRequests,
				"routed_requests":    c.RoutedRequests,
				"offloaded_requests": c.OffloadedRequests,
				"write_errors":       c.WriteErrors,
				"read_errors":        c.ReadErrors,
				"in_request":         c.InRequest,
			}
			tags := map[string]string{
				"core_id":   strconv.Itoa(c.CoreID),
				"worker_id": strconv.Itoa(w.WorkerID),
				"source":    s.source,
			}
			acc.AddFields("uwsgi_cores", fields, tags)
		}

	}
}

func init() {
	inputs.Add("uwsgi", func() cua.Input {
		return &Uwsgi{
			Timeout: internal.Duration{Duration: 5 * time.Second},
		}
	})
}

// StatsServer defines the stats server structure.
type StatsServer struct {
	// Tags
	source  string
	PID     int    `json:"pid"`
	UID     int    `json:"uid"`
	GID     int    `json:"gid"`
	Version string `json:"version"`

	// Fields
	ListenQueue       int `json:"listen_queue"`
	ListenQueueErrors int `json:"listen_queue_errors"`
	SignalQueue       int `json:"signal_queue"`
	Load              int `json:"load"`

	Workers []*Worker `json:"workers"`
}

// Worker defines the worker metric structure.
type Worker struct {
	// Tags
	WorkerID int `json:"id"`
	PID      int `json:"pid"`

	// Fields
	Accepting     int    `json:"accepting"`
	Requests      int    `json:"requests"`
	DeltaRequests int    `json:"delta_requests"`
	Exceptions    int    `json:"exceptions"`
	HarakiriCount int    `json:"harakiri_count"`
	Signals       int    `json:"signals"`
	SignalQueue   int    `json:"signal_queue"`
	Status        string `json:"status"`
	Rss           int    `json:"rss"`
	Vsz           int    `json:"vsz"`
	RunningTime   int    `json:"running_time"`
	LastSpawn     int    `json:"last_spawn"`
	RespawnCount  int    `json:"respawn_count"`
	Tx            int    `json:"tx"`
	AvgRt         int    `json:"avg_rt"`

	Apps  []*App  `json:"apps"`
	Cores []*Core `json:"cores"`
}

// App defines the app metric structure.
type App struct {
	// Tags
	AppID int `json:"id"`

	// Fields
	Modifier1   int `json:"modifier1"`
	Requests    int `json:"requests"`
	StartupTime int `json:"startup_time"`
	Exceptions  int `json:"exceptions"`
}

// Core defines the core metric structure.
type Core struct {
	// Tags
	CoreID int `json:"id"`

	// Fields
	Requests          int `json:"requests"`
	StaticRequests    int `json:"static_requests"`
	RoutedRequests    int `json:"routed_requests"`
	OffloadedRequests int `json:"offloaded_requests"`
	WriteErrors       int `json:"write_errors"`
	ReadErrors        int `json:"read_errors"`
	InRequest         int `json:"in_request"`
}
