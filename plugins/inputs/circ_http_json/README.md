# Circonus HTTP JSON

This input plugin provides the ability to fetch [Circonus HTTPTrap stream tag and structured format metrics](https://docs.circonus.com/circonus/integrations/library/json-push-httptrap/#stream-tags) and forward them to a Circonus Unified Agent check.

## Configuration

[IRONdb](https://docs.circonus.com/irondb/administration/monitoring/#json) example:

```toml
[[inputs.circ_http_json]]
  instance_id = "idb_stats"
  url = "http://127.0.0.1:8112/stats.json?format=tagged"
```

Note the addition of `?format=tagged` query argument -- use for these ironDB endpoints to ensure stream tagged, structured metric format.

## Example Metric Format

```json
 {
   "foo|ST[env:prod,app:web]": { "_type": "n", "_value": 12 },
   "foo|ST[env:qa,app:web]":   { "_type": "I", "_value": 0 },
   "foo|ST[b\"fihiYXIp\":b\"PHF1dXg+\"]": { "_type": "L", "_value": 3 }
 }
```

The metric type `_type` must be a valid HTTPTrap / Reconnoiter type:

* `i` int
* `I` uint
* `l` int64
* `L` uint64
* `n` double
* `h` histogram
* `H` cumulative histogram
* `s` text

The optional `_ts` timestamp (in milliseconds) value may also be used.
