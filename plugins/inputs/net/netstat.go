package net

import (
	"context"
	"fmt"
	"syscall"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/system"
)

type Stats struct {
	ps system.PS
}

func (*Stats) Description() string {
	return "Read TCP metrics such as established, time wait and sockets counts."
}

var tcpstatSampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)
`

func (*Stats) SampleConfig() string {
	return tcpstatSampleConfig
}

func (s *Stats) Gather(ctx context.Context, acc cua.Accumulator) error {
	netconns, err := s.ps.NetConnections()
	if err != nil {
		return fmt.Errorf("error getting net connections info: %w", err)
	}
	counts := make(map[string]int)
	counts["UDP"] = 0

	// TODO: add family to tags or else
	tags := map[string]string{}
	for _, netcon := range netconns {
		if netcon.Type == syscall.SOCK_DGRAM {
			counts["UDP"]++
			continue // UDP has no status
		}
		c, ok := counts[netcon.Status]
		if !ok {
			counts[netcon.Status] = 0
		}
		counts[netcon.Status] = c + 1
	}

	fields := map[string]interface{}{
		"tcp_established": counts["ESTABLISHED"],
		"tcp_syn_sent":    counts["SYN_SENT"],
		"tcp_syn_recv":    counts["SYN_RECV"],
		"tcp_fin_wait1":   counts["FIN_WAIT1"],
		"tcp_fin_wait2":   counts["FIN_WAIT2"],
		"tcp_time_wait":   counts["TIME_WAIT"],
		"tcp_close":       counts["CLOSE"],
		"tcp_close_wait":  counts["CLOSE_WAIT"],
		"tcp_last_ack":    counts["LAST_ACK"],
		"tcp_listen":      counts["LISTEN"],
		"tcp_closing":     counts["CLOSING"],
		"tcp_none":        counts["NONE"],
		"udp_socket":      counts["UDP"],
	}
	acc.AddFields("netstat", fields, tags)

	return nil
}

func init() {
	inputs.Add("netstat", func() cua.Input {
		return &Stats{ps: system.NewSystemPS()}
	})
}
