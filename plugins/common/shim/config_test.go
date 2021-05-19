package shim

import (
	"context"
	"os"
	"testing"
	"time"

	tgConfig "github.com/circonus-labs/circonus-unified-agent/config"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/processors"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("SECRET_TOKEN", "xxxxxxxxxx")
	os.Setenv("SECRET_VALUE", `test"\test`)

	inputs.Add("test", func() cua.Input {
		return &serviceInput{}
	})

	c := "./testdata/plugin.conf"
	conf, err := LoadConfig(&c)
	require.NoError(t, err)

	inp := conf.Input.(*serviceInput)

	require.Equal(t, "awesome name", inp.ServiceName)
	require.Equal(t, "xxxxxxxxxx", inp.SecretToken)
	require.Equal(t, `test"\test`, inp.SecretValue)
}

func TestDefaultImportedPluginsSelfRegisters(t *testing.T) {
	inputs.Add("test", func() cua.Input {
		return &testInput{}
	})

	cfg, err := LoadConfig(nil)
	require.NoError(t, err)
	require.Equal(t, "test", cfg.Input.Description())
}

func TestLoadingSpecialTypes(t *testing.T) {
	inputs.Add("test", func() cua.Input {
		return &testDurationInput{}
	})

	c := "./testdata/special.conf"
	conf, err := LoadConfig(&c)
	require.NoError(t, err)

	inp := conf.Input.(*testDurationInput)

	require.EqualValues(t, 3*time.Second, inp.Duration)
	require.EqualValues(t, 3*1000*1000, inp.Size)
}

func TestLoadingProcessorWithConfig(t *testing.T) {
	proc := &testConfigProcessor{}
	processors.Add("test_config_load", func() cua.Processor {
		return proc
	})

	c := "./testdata/processor.conf"
	_, err := LoadConfig(&c)
	require.NoError(t, err)

	require.EqualValues(t, "yep", proc.Loaded)
}

type testDurationInput struct {
	Duration tgConfig.Duration `toml:"duration"`
	Size     tgConfig.Size     `toml:"size"`
}

func (i *testDurationInput) SampleConfig() string {
	return ""
}

func (i *testDurationInput) Description() string {
	return ""
}
func (i *testDurationInput) Gather(ctx context.Context, acc cua.Accumulator) error {
	return nil
}

type testConfigProcessor struct {
	Loaded string `toml:"loaded"`
}

func (p *testConfigProcessor) SampleConfig() string {
	return ""
}

func (p *testConfigProcessor) Description() string {
	return ""
}
func (p *testConfigProcessor) Apply(metrics ...cua.Metric) []cua.Metric {
	return metrics
}
