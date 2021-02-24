package shim

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/outputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/processors"
)

type Config struct {
	Inputs     map[string][]toml.Primitive
	Processors map[string][]toml.Primitive
	Outputs    map[string][]toml.Primitive
}

type LoadedConfig struct {
	Input     cua.Input
	Processor cua.StreamingProcessor
	Output    cua.Output
}

// LoadConfig Adds plugins to the shim
func (s *Shim) LoadConfig(filePath *string) error {
	conf, err := LoadConfig(filePath)
	if err != nil {
		return err
	}
	switch {
	case conf.Input != nil:
		if err = s.AddInput(conf.Input); err != nil {
			return fmt.Errorf("Failed to add Input: %w", err)
		}
	case conf.Processor != nil:
		if err = s.AddStreamingProcessor(conf.Processor); err != nil {
			return fmt.Errorf("Failed to add Processor: %w", err)
		}
	case conf.Output != nil:
		if err = s.AddOutput(conf.Output); err != nil {
			return fmt.Errorf("Failed to add Output: %w", err)
		}
	}
	return nil
}

// LoadConfig loads the config and returns inputs that later need to be loaded.
func LoadConfig(filePath *string) (loaded LoadedConfig, err error) {
	var data string
	conf := Config{}
	if filePath != nil && *filePath != "" {

		b, err := os.ReadFile(*filePath)
		if err != nil {
			return LoadedConfig{}, fmt.Errorf("readfile (%s): %w", *filePath, err)
		}

		data = expandEnvVars(b)

	} else {
		conf, err = DefaultImportedPlugins()
		if err != nil {
			return LoadedConfig{}, err
		}
	}

	md, err := toml.Decode(data, &conf)
	if err != nil {
		return LoadedConfig{}, fmt.Errorf("toml decode: %w", err)
	}

	return createPluginsWithTomlConfig(md, conf)
}

func expandEnvVars(contents []byte) string {
	return os.Expand(string(contents), getEnv)
}

func getEnv(key string) string {
	v := os.Getenv(key)

	return envVarEscaper.Replace(v)
}

func createPluginsWithTomlConfig(md toml.MetaData, conf Config) (LoadedConfig, error) {
	loadedConf := LoadedConfig{}

	for name, primitives := range conf.Inputs {
		creator, ok := inputs.Inputs[name]
		if !ok {
			return loadedConf, errors.New("unknown input " + name)
		}

		plugin := creator()
		if len(primitives) > 0 {
			primitive := primitives[0]
			if err := md.PrimitiveDecode(primitive, plugin); err != nil {
				return loadedConf, fmt.Errorf("primitive decode: %w", err)
			}
		}

		loadedConf.Input = plugin
		break
	}

	for name, primitives := range conf.Processors {
		creator, ok := processors.Processors[name]
		if !ok {
			return loadedConf, errors.New("unknown processor " + name)
		}

		plugin := creator()
		if len(primitives) > 0 {
			primitive := primitives[0]
			var p cua.PluginDescriber = plugin
			if processor, ok := plugin.(unwrappable); ok {
				p = processor.Unwrap()
			}
			if err := md.PrimitiveDecode(primitive, p); err != nil {
				return loadedConf, fmt.Errorf("primitive decode: %w", err)
			}
		}
		loadedConf.Processor = plugin
		break
	}

	for name, primitives := range conf.Outputs {
		creator, ok := outputs.Outputs[name]
		if !ok {
			return loadedConf, fmt.Errorf("unknown output (%s)", name)
		}

		plugin := creator()
		if len(primitives) > 0 {
			primitive := primitives[0]
			if err := md.PrimitiveDecode(primitive, plugin); err != nil {
				return loadedConf, fmt.Errorf("primitive decode: %w", err)
			}
		}
		loadedConf.Output = plugin
		break
	}
	return loadedConf, nil
}

// DefaultImportedPlugins defaults to whatever plugins happen to be loaded and
// have registered themselves with the registry. This makes loading plugins
// without having to define a config dead easy.
func DefaultImportedPlugins() (Config, error) {
	conf := Config{
		Inputs:     map[string][]toml.Primitive{},
		Processors: map[string][]toml.Primitive{},
		Outputs:    map[string][]toml.Primitive{},
	}
	for name := range inputs.Inputs {
		log.Println("No config found. Loading default config for plugin", name)
		conf.Inputs[name] = []toml.Primitive{}
		return conf, nil
	}
	for name := range processors.Processors {
		log.Println("No config found. Loading default config for plugin", name)
		conf.Processors[name] = []toml.Primitive{}
		return conf, nil
	}
	for name := range outputs.Outputs {
		log.Println("No config found. Loading default config for plugin", name)
		conf.Outputs[name] = []toml.Primitive{}
		return conf, nil
	}
	return conf, nil
}

type unwrappable interface {
	Unwrap() cua.Processor
}
