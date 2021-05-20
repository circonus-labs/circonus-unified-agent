// +build linux

package kernelvmstat

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type KernelVmstat struct {
	statFile string
}

func (k *KernelVmstat) Description() string {
	return "Get kernel statistics from /proc/vmstat"
}

func (k *KernelVmstat) SampleConfig() string {
	return ""
}

func (k *KernelVmstat) Gather(ctx context.Context, acc cua.Accumulator) error {
	data, err := k.getProcVmstat()
	if err != nil {
		return err
	}

	fields := make(map[string]interface{})

	dataFields := bytes.Fields(data)
	for i, field := range dataFields {

		// dataFields is an array of {"stat1_name", "stat1_value", "stat2_name",
		// "stat2_value", ...}
		// We only want the even number index as that contain the stat name.
		if i%2 == 0 {
			// Convert the stat value into an integer.
			m, err := strconv.ParseInt(string(dataFields[i+1]), 10, 64)
			if err != nil {
				return fmt.Errorf("parseint (%s): %w", string(dataFields[i+1]), err)
			}

			fields[string(field)] = m
		}
	}

	acc.AddFields("kernel_vmstat", fields, map[string]string{})
	return nil
}

func (k *KernelVmstat) getProcVmstat() ([]byte, error) {
	if _, err := os.Stat(k.statFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("kernel_vmstat: %s does not exist", k.statFile)
	} else if err != nil {
		return nil, fmt.Errorf("kernal_vmstat (%s): %w", k.statFile, err)
	}

	data, err := os.ReadFile(k.statFile)
	if err != nil {
		return nil, fmt.Errorf("kernel_vmstat (read %s): %w", k.statFile, err)
	}

	return data, nil
}

func init() {
	inputs.Add("kernel_vmstat", func() cua.Input {
		return &KernelVmstat{
			statFile: "/proc/vmstat",
		}
	})
}
