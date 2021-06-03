# Circonus

The `circonus` output data format converts metrics into Circonus JSON documents.

## Configuration

```toml
[[outputs.file]]
  ## Files to write to, "stdout" is a specially handled file.
  files = ["stdout", "/tmp/metrics.out"]

  ## Data format to output.
  ## Each data format has its own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/circonus-labs/circonus-unified-agent/blob/master/docs/DATA_FORMATS_OUTPUT.md
  data_format = "circonus"

  ## The resolution to use for the metric timestamp.  Must be a duration string
  ## such as "1ns", "1us", "1ms", "10ms", "1s".  Durations are truncated to
  ## the power of 10 less than the specified units.
  json_timestamp_units = "1s"
```

## Examples

Standard form:

```json
{"total_alloc_bytes|ST[input_metric_group:internal_memstats,input_plugin:internal]": {"_value":12509192, "_type": "L", "_ts":1622727015000}}
```

When an output plugin needs to emit multiple metrics at one time, it may use
the batch format.  The use of batch format is determined by the plugin,
reference the documentation for the specific plugin.

```json
{"total|ST[input_plugin:swap]": {"_value":0, "_type": "L", "_ts":1622727015000}}
{"used|ST[input_plugin:swap]": {"_value":0, "_type": "L", "_ts":1622727015000}}
{"free|ST[input_plugin:swap]": {"_value":0, "_type": "L", "_ts":1622727015000}}
{"used_percent|ST[input_plugin:swap]": {"_value":0, "_type": "n", "_ts":1622727015000}}
{"in|ST[input_plugin:swap]": {"_value":0, "_type": "L", "_ts":1622727015000}}
{"out|ST[input_plugin:swap]": {"_value":0, "_type": "L", "_ts":1622727015000}}
```
