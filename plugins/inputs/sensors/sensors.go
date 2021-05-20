// +build linux

package sensors

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

var (
	execCommand    = exec.Command // execCommand is used to mock commands in tests.
	numberRegp     = regexp.MustCompile("[0-9]+")
	defaultTimeout = internal.Duration{Duration: 5 * time.Second}
)

type Sensors struct {
	RemoveNumbers bool              `toml:"remove_numbers"`
	Timeout       internal.Duration `toml:"timeout"`
	path          string
}

func (*Sensors) Description() string {
	return "Monitor sensors, requires lm-sensors package"
}

func (*Sensors) SampleConfig() string {
	return `
  ## Remove numbers from field names.
  ## If true, a field name like 'temp1_input' will be changed to 'temp_input'.
  # remove_numbers = true

  ## Timeout is the maximum amount of time that the sensors command can run.
  # timeout = "5s"
`

}

func (s *Sensors) Gather(ctx context.Context, acc cua.Accumulator) error {
	if len(s.path) == 0 {
		return errors.New("sensors not found: verify that lm-sensors package is installed and that sensors is in your PATH")
	}

	return s.parse(acc)
}

// parse forks the command:
//     sensors -u -A
// and parses the output to add it to the cua.Accumulator.
func (s *Sensors) parse(acc cua.Accumulator) error {
	tags := map[string]string{}
	fields := map[string]interface{}{}
	chip := ""
	cmd := execCommand(s.path, "-A", "-u")
	out, err := internal.StdOutputTimeout(cmd, s.Timeout.Duration)
	if err != nil {
		return fmt.Errorf("failed to run command %s: %w - %s", strings.Join(cmd.Args, " "), err, string(out))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			acc.AddFields("sensors", fields, tags)
			chip = ""
			tags = map[string]string{}
			fields = map[string]interface{}{}
			continue
		}
		if len(chip) == 0 {
			chip = line
			tags["chip"] = chip
			continue
		}
		if !strings.HasPrefix(line, "  ") {
			if len(tags) > 1 {
				acc.AddFields("sensors", fields, tags)
			}
			fields = map[string]interface{}{}
			tags = map[string]string{
				"chip":    chip,
				"feature": strings.TrimRight(snake(line), ":"),
			}
		} else {
			splitted := strings.Split(line, ":")
			fieldName := strings.TrimSpace(splitted[0])
			if s.RemoveNumbers {
				fieldName = numberRegp.ReplaceAllString(fieldName, "")
			}
			fieldValue, err := strconv.ParseFloat(strings.TrimSpace(splitted[1]), 64)
			if err != nil {
				return fmt.Errorf("sensors parsefloat (%s): %w", strings.TrimSpace(splitted[1]), err)
			}
			fields[fieldName] = fieldValue
		}
	}
	acc.AddFields("sensors", fields, tags)
	return nil
}

// snake converts string to snake case
func snake(input string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(input), " ", "_"))
}

func init() {
	s := Sensors{
		RemoveNumbers: true,
		Timeout:       defaultTimeout,
	}
	path, _ := exec.LookPath("sensors")
	if len(path) > 0 {
		s.path = path
	}
	inputs.Add("sensors", func() cua.Input {
		return &s
	})
}
