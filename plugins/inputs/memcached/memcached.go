package memcached

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

// Memcached is a memcached plugin
type Memcached struct {
	Servers     []string
	UnixSockets []string
	lastVerSend *time.Time
}

var sampleConfig = `
  ## An array of address to gather stats about. Specify an ip on hostname
  ## with optional port. ie localhost, 10.0.0.1:11211, etc.
  servers = ["localhost:11211"]
  # unix_sockets = ["/var/run/memcached.sock"]
`

var defaultTimeout = 5 * time.Second

// The list of metrics that should be sent
// https://github.com/memcached/memcached/blob/master/doc/protocol.txt#L1325 (the "Settings statistics" section)
var sendMetrics = []string{
	"uptime",
	"version",
	"curr_items",
	"total_items",
	"bytes",
	"max_connections",
	"curr_connections",
	"total_connections",
	"rejected_connecctions",
	"connection_structures",
	"response_obj_oom",
	"response_obj_count",
	"response_obj_bytes",
	"read_buf_count",
	"read_buf_bytes",
	"read_buf_bytes_free",
	"read_buf_oom",
	"cmd_get",
	"cmd_set",
	"cmd_flush",
	"cmd_touch",
	"get_hits",
	"get_misses",
	"get_expired",
	"get_flushed",
	"delete_misses",
	"delete_hits",
	"incr_misses",
	"incr_hits",
	"decr_misses",
	"decr_hits",
	"cas_hits",
	"cas_misses",
	"cas_badval",
	"touch_hits",
	"touch_misses",
	"auth_cmds",
	"auth_errors",
	"idle_kicks",
	"evictions",
	"reclaimed",
	"bytes_read",
	"bytes_written",
	"limit_maxbytes",
	"accepting_conns",
	"listen_disabled_num",
	"time_in_listen_disabled",
	"threads",
	"conn_yields",
	"hash_power_level",
	"hash_bytes",
	"hash_is_expanding",
	"expired_unfetched",
	"evicted_unfetched",
	"evicted_active",
	"slabs_reassign_running",
	"slabs_moved",
	"crawler_reclaimed",
	"crawler_items_checked",
	"lrutail_reflocked",
	"moves_to_cold",
	"moves_to_warm",
	"moves_within_lru",
	"direct_reclaims",
	"lru_crawler_starts",
	"lru_maintainer_juggles",
	"slab_global_page_pool",
	"slab_reassign_rescues",
	"slab_reassign_evictions_nomem",
	"slab_reassign_chunk_rescues",
	"slab_reassign_inline_reclaim",
	"slab_reassign_busy_deletes",
	"log_worker_dropped",
	"log_worker_written",
	"log_watcher_skipped",
	"log_watcher_sent",
	"unexpected_napi_ids",
	"round_robin_fallback",
}

// SampleConfig returns sample configuration message
func (m *Memcached) SampleConfig() string {
	return sampleConfig
}

// Description returns description of Memcached plugin
func (m *Memcached) Description() string {
	return "Read metrics from one or many memcached servers"
}

// Gather reads stats from all configured servers accumulates stats
func (m *Memcached) Gather(ctx context.Context, acc cua.Accumulator) error {
	if len(m.Servers) == 0 && len(m.UnixSockets) == 0 {
		return m.gatherServer(":11211", false, acc)
	}

	for _, serverAddress := range m.Servers {
		acc.AddError(m.gatherServer(serverAddress, false, acc))
	}

	for _, unixAddress := range m.UnixSockets {
		acc.AddError(m.gatherServer(unixAddress, true, acc))
	}

	return nil
}

func (m *Memcached) gatherServer(
	address string,
	unix bool,
	acc cua.Accumulator,
) error {
	var conn net.Conn
	var err error
	if unix {
		conn, err = net.DialTimeout("unix", address, defaultTimeout)
		if err != nil {
			return fmt.Errorf("dial timeout unix (%s): %w", address, err)
		}
		defer conn.Close()
	} else {
		_, _, err = net.SplitHostPort(address)
		if err != nil {
			address += ":11211"
		}

		conn, err = net.DialTimeout("tcp", address, defaultTimeout)
		if err != nil {
			return fmt.Errorf("dial timeout tcp (%s): %w", address, err)
		}
		defer conn.Close()
	}

	if conn == nil {
		return fmt.Errorf("Failed to create net connection")
	}

	// Extend connection
	_ = conn.SetDeadline(time.Now().Add(defaultTimeout))

	// Read and write buffer
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	// Send command
	if _, err := fmt.Fprint(rw, "stats\r\n"); err != nil {
		return fmt.Errorf("send cmd: %w", err)
	}
	if err := rw.Flush(); err != nil {
		return fmt.Errorf("bufio flush: %w", err)
	}

	values, err := parseResponse(rw.Reader)
	if err != nil {
		return err
	}

	// Add server address as a tag
	tags := map[string]string{"server": address}

	// Process values
	fields := make(map[string]interface{})
	for _, key := range sendMetrics {
		if value, ok := values[key]; ok {
			if key == "version" {
				if m.lastVerSend == nil || time.Since(*m.lastVerSend) > 5*time.Minute {
					fields[key] = value
					t := time.Now().UTC()
					m.lastVerSend = &t
				}
			}
			// Mostly it is the number
			if iValue, errParse := strconv.ParseInt(value, 10, 64); errParse == nil {
				fields[key] = iValue
			} else {
				fields[key] = value
			}
		}
	}
	acc.AddFields("memcached", fields, tags)
	return nil
}

func parseResponse(r *bufio.Reader) (map[string]string, error) {
	values := make(map[string]string)

	for {
		// Read line
		line, _, errRead := r.ReadLine()
		if errRead != nil {
			return values, fmt.Errorf("bufio read: %w", errRead)
		}
		// Done
		if bytes.Equal(line, []byte("END")) {
			break
		}
		// Read values
		s := bytes.SplitN(line, []byte(" "), 3)
		if len(s) != 3 || !bytes.Equal(s[0], []byte("STAT")) {
			return values, fmt.Errorf("unexpected line in stats response: %q", line)
		}

		// Save values
		values[string(s[1])] = string(s[2])
	}
	return values, nil
}

func init() {
	inputs.Add("memcached", func() cua.Input {
		return &Memcached{}
	})
}
