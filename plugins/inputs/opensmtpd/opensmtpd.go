package opensmtpd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/filter"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type runner func(cmdName string, Timeout internal.Duration, UseSudo bool) (*bytes.Buffer, error)

// Opensmtpd is used to store configuration values
type Opensmtpd struct {
	Binary  string
	Timeout internal.Duration
	UseSudo bool

	// filter filter.Filter
	run runner
}

var defaultBinary = "/usr/sbin/smtpctl"
var defaultTimeout = internal.Duration{Duration: time.Second}

var sampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)

  ## If running as a restricted user you can prepend sudo for additional access:
  #use_sudo = false

  ## The default location of the smtpctl binary can be overridden with:
  binary = "/usr/sbin/smtpctl"

  ## The default timeout of 1000ms can be overridden with (in milliseconds):
  timeout = 1000
`

func (s *Opensmtpd) Description() string {
	return "A plugin to collect stats from Opensmtpd - a validating, recursive, and caching DNS resolver "
}

// SampleConfig displays configuration instructions
func (s *Opensmtpd) SampleConfig() string {
	return sampleConfig
}

// Shell out to opensmtpd_stat and return the output
func opensmtpdRunner(cmdName string, timeout internal.Duration, useSudo bool) (*bytes.Buffer, error) {
	cmdArgs := []string{"show", "stats"}

	cmd := exec.Command(cmdName, cmdArgs...)

	if useSudo {
		cmdArgs = append([]string{cmdName}, cmdArgs...)
		cmd = exec.Command("sudo", cmdArgs...)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	err := internal.RunTimeout(cmd, timeout.Duration)
	if err != nil {
		return &out, fmt.Errorf("error running smtpctl: %w", err)
	}

	return &out, nil
}

// Gather collects the configured stats from smtpctl and adds them to the
// Accumulator
//
// All the dots in stat name will replaced by underscores. Histogram statistics will not be collected.
func (s *Opensmtpd) Gather(ctx context.Context, acc cua.Accumulator) error {
	// Always exclude uptime.human statistics
	statExcluded := []string{"uptime.human"}
	filterExcluded, err := filter.Compile(statExcluded)
	if err != nil {
		return fmt.Errorf("filter compile: %w", err)
	}

	out, err := s.run(s.Binary, s.Timeout, s.UseSudo)
	if err != nil {
		return fmt.Errorf("error gathering metrics: %w", err)
	}

	// Process values
	fields := make(map[string]interface{})
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {

		cols := strings.Split(scanner.Text(), "=")

		// Check split correctness
		if len(cols) != 2 {
			continue
		}

		stat := cols[0]
		value := cols[1]

		// Filter value
		if filterExcluded.Match(stat) {
			continue
		}

		field := strings.ReplaceAll(stat, ".", "_")

		fields[field], err = strconv.ParseFloat(value, 64)
		if err != nil {
			acc.AddError(fmt.Errorf("expected numerical value for %s = %v", stat, value))
		}
	}

	acc.AddFields("opensmtpd", fields, nil)

	return nil
}

func init() {
	inputs.Add("opensmtpd", func() cua.Input {
		return &Opensmtpd{
			run:     opensmtpdRunner,
			Binary:  defaultBinary,
			Timeout: defaultTimeout,
			UseSudo: false,
		}
	})
}
