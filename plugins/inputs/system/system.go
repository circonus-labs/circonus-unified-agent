package system

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
)

type Stats struct {
	Log cua.Logger
}

func (*Stats) Description() string {
	return "Read metrics about system load & uptime"
}

func (*Stats) SampleConfig() string {
	return `
  ## Uncomment to remove deprecated metrics.
  # fielddrop = ["uptime_format"]
`
}

func (s *Stats) Gather(acc cua.Accumulator) error {
	loadavg, err := load.Avg()
	if err != nil && !strings.Contains(err.Error(), "not implemented") {
		return fmt.Errorf("load avg: %w", err)
	}

	numCPUs, err := cpu.Counts(true)
	if err != nil {
		return fmt.Errorf("cpu counts: %w", err)
	}

	fields := map[string]interface{}{
		"load1":  loadavg.Load1,
		"load5":  loadavg.Load5,
		"load15": loadavg.Load15,
		"n_cpus": numCPUs,
	}

	users, err := host.Users()
	switch {
	case err == nil:
		fields["n_users"] = len(users)
	case os.IsNotExist(err):
		s.Log.Debugf("Reading users: %s", err.Error())
	case os.IsPermission(err):
		s.Log.Debug(err.Error())
	}

	now := time.Now()
	acc.AddGauge("system", fields, nil, now)

	uptime, err := host.Uptime()
	if err != nil {
		return fmt.Errorf("uptime: %w", err)
	}

	acc.AddCounter("system", map[string]interface{}{
		"uptime": uptime,
	}, nil, now)
	acc.AddFields("system", map[string]interface{}{
		"uptime_format": formatUptime(uptime),
	}, nil, now)

	return nil
}

func formatUptime(uptime uint64) string {
	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)

	days := uptime / (60 * 60 * 24)

	if days != 0 {
		s := ""
		if days > 1 {
			s = "s"
		}
		fmt.Fprintf(w, "%d day%s, ", days, s)
	}

	minutes := uptime / 60
	hours := minutes / 60
	hours %= 24
	minutes %= 60

	fmt.Fprintf(w, "%2d:%02d", hours, minutes)

	w.Flush()
	return buf.String()
}

func init() {
	inputs.Add("system", func() cua.Input {
		return &Stats{}
	})
}
