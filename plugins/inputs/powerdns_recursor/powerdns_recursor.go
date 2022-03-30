package powerdnsrecursor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type PowerdnsRecursor struct {
	UnixSockets []string `toml:"unix_sockets"`
	SocketDir   string   `toml:"socket_dir"`
	SocketMode  string   `toml:"socket_mode"`

	mode uint32
}

var defaultTimeout = 5 * time.Second

var sampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)

  ## Path to the Recursor control socket.
  unix_sockets = ["/var/run/pdns_recursor.controlsocket"]

  ## Directory to create receive socket.  This default is likely not writable,
  ## please reference the full plugin documentation for a recommended setup.
  # socket_dir = "/var/run/"
  ## Socket permissions for the receive socket.
  # socket_mode = "0666"
`

func (p *PowerdnsRecursor) SampleConfig() string {
	return sampleConfig
}

func (p *PowerdnsRecursor) Description() string {
	return "Read metrics from one or many PowerDNS Recursor servers"
}

func (p *PowerdnsRecursor) Init() error {
	if p.SocketMode != "" {
		mode, err := strconv.ParseUint(p.SocketMode, 8, 32)
		if err != nil {
			return fmt.Errorf("could not parse socket_mode: %w", err)
		}

		p.mode = uint32(mode)
	}
	return nil
}

func (p *PowerdnsRecursor) Gather(ctx context.Context, acc cua.Accumulator) error {
	if len(p.UnixSockets) == 0 {
		return p.gatherServer("/var/run/pdns_recursor.controlsocket", acc)
	}

	for _, serverSocket := range p.UnixSockets {
		if err := p.gatherServer(serverSocket, acc); err != nil {
			acc.AddError(err)
		}
	}

	return nil
}

func (p *PowerdnsRecursor) gatherServer(address string, acc cua.Accumulator) error {
	randomNumber := rand.Int63() //nolint:gosec // G404
	recvSocket := filepath.Join("/", "var", "run", fmt.Sprintf("pdns_recursor_cua%d", randomNumber))
	if p.SocketDir != "" {
		recvSocket = filepath.Join(p.SocketDir, fmt.Sprintf("pdns_recursor_cua%d", randomNumber))
	}

	laddr, err := net.ResolveUnixAddr("unixgram", recvSocket)
	if err != nil {
		return fmt.Errorf("resolve (%s): %w", recvSocket, err)
	}
	defer os.Remove(recvSocket)
	raddr, err := net.ResolveUnixAddr("unixgram", address)
	if err != nil {
		return fmt.Errorf("resolve (%s): %w", address, err)
	}
	conn, err := net.DialUnix("unixgram", laddr, raddr)
	if err != nil {
		return fmt.Errorf("dial (%s  %s): %w", laddr.String(), raddr.String(), err)
	}
	if err := os.Chmod(recvSocket, os.FileMode(p.mode)); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(defaultTimeout))

	// Read and write buffer
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	// Send command
	if _, err := fmt.Fprint(rw, "get-all\n"); err != nil {
		return nil
	}
	if err := rw.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}

	// Read data
	buf := make([]byte, 16384)
	n, err := rw.Read(buf)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if n == 0 {
		return errors.New("no data received")
	}

	metrics := string(buf)

	// Process data
	fields := parseResponse(metrics)

	// Add server socket as a tag
	tags := map[string]string{"server": address}

	acc.AddFields("powerdns_recursor", fields, tags)

	conn.Close()

	return nil
}

func parseResponse(metrics string) map[string]interface{} {
	values := make(map[string]interface{})

	s := strings.Split(metrics, "\n")

	for _, metric := range s[:len(s)-1] {
		m := strings.Split(metric, "\t")
		if len(m) < 2 {
			continue
		}

		i, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			log.Printf("E! [inputs.powerdns_recursor] error parsing integer for metric %q: %s",
				metric, err.Error())
			continue
		}
		values[m[0]] = i
	}

	return values
}

func init() {
	inputs.Add("powerdns_recursor", func() cua.Input {
		return &PowerdnsRecursor{
			mode: uint32(0666),
		}
	})
}
