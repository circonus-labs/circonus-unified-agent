# Circonus Output Plugin

This plugin writes metrics data to the Circonus platform. In order to use this
plugin, an HTTPTrap check must be configured on a Circonus broker. This check
can be automatically created by the plugin or manually configured (see the
plugin configuration information). For information about Circonus HTTPTrap
check configuration click [here][docs].

## Configuration

```toml
[[outputs.circonus]]
  ## Circonus API token must be provided to use this plugin:
  api_token = ""

  ## Circonus API application (associated with token):
  ## example:
  # api_app = "circonus-unified-agent"

  ## Circonus API URL:
  ## example:
  # api_url = "https://api.circonus.com/"

  ## Circonus API TLS CA file, optional, for internal deployments with private certificates: 
  ## example:
  # api_tls_ca = "/opt/circonus/unified-agent/etc/circonus_api_ca.pem"

  ## Broker
  ## Optional: explicit broker id or blank (default blank, auto select)
  ## example:
  # broker = "/broker/35"

  ## Allow snmp trap text event metrics to flow through to circonus.
  ## This is off by default, and snmp trap text events will be dropped.
  ## Enabling this will result in increased billing costs.
  # allow_snmp_trap_events = false

  ## Sub output - is this an additional output to handle specific plugin metrics (e.g. not the main, host system output)
  ## Optional - if multiple outputs think they are the main, there can be duplicate metric submissions
  # sub_output = false

  ## Pool size - controls the number of batch processors
  ## Optional: mostly applicable to large number of inputs or inputs producing lots (100K+) of metrics
  # pool_size = 2

```

### Configuration Options

|Setting|Description|
|-------|-----------|
|`api_token`|The authentication token to used when connecting to the Circonus API. It is recommended to create a token/application combination specifically for use with this plugin. This is required.|
|`api_url`|The URL that can be used to connect to the Circonus API. This will default to the Circonus SaaS API URL if not provided.|
|`api_app`|The API token application to use when connecting to the Circonus API. This will default to `circonus-unified-agent` if not provided.|
|`api_tls_ca`|The certificate authority file to use when connecting to the Circonus API, if needed.|
|`broker`|The CID of a Circonus broker to use when automatically creating a check. If omitted, then a random eligible broker will be selected.|
|`pool_size`|Optional: size of the processor pool for a given output instance - default 2.|
|`cache_configs`|Optional: cache check bundle configurations - efficient for large number of inputs - default false.|
|`cache_dir`|Optional: where to cache the check bundle configurations - must be read/write for user running cua - default "".|
|`allow_snmp_trap_events`|Optional: send snmp_trap text events to circonus - may result in high billing costs - default false.|
|`sub_output`|A dedicated, special purpose, output, don't send internal cua metrics, etc. Use this when routing specific metrics to an additional instance of the Circonus output plugin.|

[docs]: https://docs.circonus.com/circonus/checks/check-types/httptrap
