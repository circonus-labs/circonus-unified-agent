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

  ## Check name prefix - unique prefix to use for all checks created by this instance
  ## default is the hostname from the OS. If set, "host" tag on metrics will be 
  ## overridden with this value. For containers, use omit_hostname=true in agent section
  ## and set this value, so that the plugin will be able to predictively find the check 
  ## for this instance. Otherwise, the container's os.Hostname() will be used
  ## (resulting in a new check being created every time the container starts).
  ## example:
  # check_name_prefix = "example"

  ## One check - all metrics go to a single check vs one check per input plugin
  ## NOTE: this effectively disables automatic dashboards for supported plugins
  # one_check = false
  
  ## Broker
  ## Optional: explicit broker id or blank (default blank, auto select)
  ## example:
  # broker = "/broker/35"
```

### Configuration Options

|Setting|Description|
|-------|-----------|
|`api_token`|The authentication token to used when connecting to the Circonus API. It is recommended to create a token/application combination specifically for use with this plugin. This is required.|
|`api_url`|The URL that can be used to connect to the Circonus API. This will default to the Circonus SaaS API URL if not provided.|
|`api_app`|The API token application to use when connecting to the Circonus API. This will default to `circonus-unified-agent` if not provided.|
|`api_tls_ca`|The certificate authority file to use when connecting to the Circonus API, if needed.|
|`check_name_prefix`|Unique prefix to use for all checks created by this instance. Default is the host name from the OS.|
|`one_check`|Send all metrics to one single check. Default is one check per active plugin.|
|`broker`|The CID of a Circonus broker to use when automatically creating a check. If omitted, then a random eligible broker will be selected.|

[docs]: https://docs.circonus.com/circonus/checks/check-types/httptrap
