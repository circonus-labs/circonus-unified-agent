### Output Plugins

This section is for developers who want to create a new output sink. Outputs
are created in a similar manner as collection plugins, and their interface has
similar constructs.

### Output Plugin Guidelines

- An output must conform to the [cua.Output][] interface.
- Outputs should call `outputs.Add` in their `init` function to register
  themselves.  See below for a quick example.
- To be available within the agent itself, plugins must add themselves to the
  `github.com/circonus-labs/circonus-unified-agent/plugins/outputs/all/all.go` file.
- The `SampleConfig` function should return valid toml that describes how the
  plugin can be configured. This is included in `circonus-unified-agentd config`.  Please
  consult the [SampleConfig][] page for the latest style guidelines.
- The `Description` function should say in one line what this output does.
- Follow the recommended [CodeStyle][].

### Output Plugin Example

```go
package simpleoutput

// simpleoutput.go

import (
    "github.com/circonus-labs/circonus-unified-agent/cua"
    "github.com/circonus-labs/circonus-unified-agent/outputs"
)

type Simple struct {
    Ok  bool       `toml:"ok"`
    Log cua.Logger `toml:"-"`
}

func (s *Simple) Description() string {
    return "a demo output"
}

func (s *Simple) SampleConfig() string {
    return `
  ok = true
`
}

// Init is for setup, and validating config.
func (s *Simple) Init() error {
    return nil
}

func (s *Simple) Connect() error {
    // Make any connection required here
    return nil
}

func (s *Simple) Close() error {
    // Close any connections here.
    // Write will not be called once Close is called, so there is no need to synchronize.
    return nil
}

// Write should write immediately to the output, and not buffer writes
// (the agent manages the buffer for you). Returning an error will fail this
// batch of writes and the entire batch will be retried automatically.
func (s *Simple) Write(metrics []cua.Metric) (int, error) {
    for _, metric := range metrics {
        // write `metric` to the output sink here
    }
    return 0, nil
}

func init() {
    outputs.Add("simpleoutput", func() cua.Output { return &Simple{} })
}

```

## Data Formats

Some output plugins, such as the [file][] plugin, can write in any supported
[output data formats][].

In order to enable this, you must specify a
`SetSerializer(serializer serializers.Serializer)`
function on the plugin object (see the file plugin for an example), as well as
defining `serializer` as a field of the object.

You can then utilize the serializer internally in your plugin, serializing data
before it's written. The agent's configuration layer will take care of
instantiating and creating the `Serializer` object.

You should also add the following to your `SampleConfig()`:

```toml
  ## Data format to output.
  ## Each data format has its own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/circonus-labs/circonus-unified-agent/blob/master/docs/DATA_FORMATS_OUTPUT.md
  data_format = "influx"
```

## Flushing Metrics to Outputs

Metrics are flushed to outputs when any of the following events happen:

- `flush_interval + rand(flush_jitter)` has elapsed since start or the last flush interval
- At least `metric_batch_size` count of metrics are waiting in the buffer
- The agent process has received a SIGUSR1 signal

Note that if the flush takes longer than the `agent.interval` to write the metrics
to the output, you'll see a message saying the output `did not complete within its
flush interval`. This may mean your output is not keeping up with the flow of metrics,
and you may want to look into enabling compression, reducing the size of your metrics,
or investigate other reasons why the writes might be taking longer than expected.

[file]: https://github.com/circonus-labs/circonus-unified-agent/tree/master/plugins/inputs/file
[output data formats]: https://github.com/circonus-labs/circonus-unified-agent/blob/master/docs/DATA_FORMATS_OUTPUT.md
[SampleConfig]: https://github.com/circonus-labs/circonus-unified-agent/wiki/SampleConfig
[CodeStyle]: https://github.com/circonus-labs/circonus-unified-agent/wiki/CodeStyle
[cua.Output]: https://godoc.org/github.com/circonus-labs/circonus-unified-agent#Output
