# v0.0.26

* upd: (snmp) one input to handle both dm and regular output
* add: (snmp) text metric capability for syntax integer
* add: (circmgr) same check name prefix handling as regular circ output
* upd: (circmgr) use plugin and instance in cache file names
* upd: (circmgr) reduce verbosity in new metric dest err msg
* add: (circout) timestamp to agent metric
* add: (statsd) new dm statsd input
* add: (agent) context to input Start
* add: (circmgr) global tags for dm inputs
* upd: dependencies (go-trapcheck, go-trapmetrics)
* fix: lint issues
* upd: lint

# v0.0.25

* upd: enable default linux plugins for darwin and freebsd

# v0.0.24

* add: `snmp_dm` snmp input plugin with direct metrics (sends directly to circonus) for large number of plugin instances

# v0.0.23

* add: `stackdriver_circonus` input plugin to support Stackdriver dashboards in UI

# v0.0.22

> NOTE: portions of circonus configuration migrated to `[agent.circonus]` section -- see example configurations in `etc/`

* upd: deps (go-trapcheck, go-trapmetrics)
* doc: update readme
* upd: circonus output to use internal cdmd package
* add: circonus metric destination mgmt package to internal
* add: agent.circonus example
* add: agent.circonus config
* upd: error msg on cmdm init
* add: ignore test tracing dir
* fix: minor typo in install.sh
* doc: ping plugin readme, reflect default cua install path
* upd: upgrading snmp from `soniah/gosnmp` to `gosnmp/gosnmp`
* upd: modify the snmp_trap receiver functionality to better suit metrics 2.0 style ingestion
* add: initialize internal circonus module
* add: internal circonus module
* add: global circonus config
* upd: removing soniah/gosnmp and replacing with gosnmp/gosnmp
* upd: replace gosnmp
* upd: changing the way that snmp_trap handles traps to better suit circonus ingestion
* upd: updating snmp_trap from soniah/gosnmp to gosnmp/gosnmp
* upd: dep psutil v3

# v0.0.21

* upd: only update metric counter for tracking metrics for non-sub output
* add: latency metrics for metric processing
* add: submit timestamp based on metric timestamp
* add: circonus debugging specific cgm check
* fix: cache dir check
* upd: testing cache directory
* add: new memcached metrics
* add: apiclient max retries, min delay, max delay settings to address duplicate checks
* upd: dependency (cgm, go-apiclient)
* add: batch processing pool
* add: `pool_size` config option
* add: check config caching
* add: `cache_configs` and `cache_dir` options
* add: `dump_cgm_metrics` option for debugging
* add: `sub_output` option to tell 2-n instances of circonus output to not initialize default checks
* add: `dynamic_submit` option to let cgm submit metrics on its internal cadence
* add: `dynamic_interval` option to set the cgm interval
* fix: overwrite existing numeric rather than add - duplicate metrics in same batch
* add: back alias tag for multiple instances
* add: comment on limitnofile for 'too many open files' errors with large number of inputs
* fix: load defaults once per agent run, not on config reload
* upd: use instance_id for alias on inputs if alias not set (for logging)
* upd: lint config
* upd: add `input_id` to all inputs in sample config
* upd: debug messages
* upd: remove commented code
* upd: switch version parser
* upd: snmp dynamic textual conversion for tags
* add: snmp automatic tag lookups
* add: additional counters to the windows_perf_counters input plugin for Windows host monitoring service dashboard

# v0.0.20

* add: darwin build back in temporarily
* upd: clarify batch write msg with number of distinct metrics
* upd: outputs return num distinct metrics written

# v0.0.19

* upd : dependencies (cgm, go-apiclient)
* add: search tag to cgm config
* fix: no short-circuit for no default plugins

# v0.0.18

* upd: disable creating host check if no default plugins for platform
* fix: windows archive should use zip
* doc: update circonus plugin docs

# v0.0.17

* upd: changes for dashboard and support older rabbitmq vers lack of metrics
* upd: zfs and rabbitmq
* add: semver
* upd: turn on pool metrics by default
* add: support text metrics mixed with numerics

# v0.0.16

* add: simple installer script (rpm el7,el8 & deb u18,u20)
* upd: disable deprecated linter
* fix: lint issues
* upd: lint v1.38
* upd: disable default plugins in containers
* upd: ignore vagrant testing files
* upd: collection interval 60s

# v0.0.15

* upd: rearrange checks default,host,agent

# v0.0.14

* upd: switch back to metric origin for check type
* add: `input_metric_group` tag for plugins which produce multiple groups of metrics
* upd: refactor tagging and metric dest methods to use the metric struct directly
* fix: typo in rollup tag for memstats `__rollup` missing underscore

# v0.0.13

* upd: replace deprecated ioutil methods
* upd: go1.16
* fix: lint issues
* upd: build/lint configs
* upd: dependencies (cgm, golangci-lint)
* upd: use metric.Name() for check type
* upd: remove ':' between plugin and instance id in check display name
* upd: only use default check if plugin AND instance are "defaults"
* add: defaultInstanceID support

# v0.0.12

* add: cgm interval as setting
* fix: check for err != nil for abnormal exit msg
* upd: switch md5->sha256 (filestat)
* fix: lint errors
* add: additional linters
* upd: check type `httptrap:cua:plugin:os`
* upd: check display name: `host plugin (os)`
* add: emit cua version every 5m
* upd: lint fixes (gofmt/golint)
* upd: dep (sarama)
* upd: enable percpu in default cpu plugin
* fix: lint (GOOS=windows)
* fix: lint (GOOS=linux)
* fix: lint (errorlint)
* add: additional linters
* upd: tool version

# v0.0.11

* fix: cumulative histogram submission and honor metricKind settings (stackdriver)
* add: a tag for metric_kind (stackdriver)

# v0.0.10

* fix: health output missing in all

# v0.0.9

* fix: lint error
* upd: pem location
* add: health output plugin
* upd: turn off metric type debug msgs
* upd: mv queued metrics msg to debug
* upd: dep (cgm)

# v0.0.8

* upd: refactor metric debug above cgm add to catch metrics causing any errors
* add: support for non-cumulative histograms into stackdriver input plugin
* upd: remove unused templates
* fix: add 's' to project in metric descriptor request (stackdriver)
* upd: log metric queued for circ
* add: net as a default input
* upd: syntax change in tool

# v0.0.7

* upd: refactor default plugin handling in prep to support more platforms
* upd: remove darwin build target
* upd: rename example config file
* upd: dependency gopsutil
* upd: support custom api ca cert loading (circonus output)
* fix: include example configuration files
* upd: dest guard for agent version
* fix: env var syntax in example conf

# v0.0.6

* upd: remove agent version tag from internal metrics (to reduce cardinality)
* add: `cua_version` metric

# v0.0.5

* add: windows default plugin placeholder
* add: rollup tag to internal metrics

# v0.0.4

* add: debug_metrics setting
* fix: copy metric for origin and origin instance
* upd: default plugins in example configuration
* fix: ldflags typo

# v0.0.3

* add: metric volume internal metric `cua_metrics_sent`
* upd: cgm debug logging to use info
* add: default plugins go to default check (cpu,mem,disk,diskio,swap,system,kernel,processes,internal)

# v0.0.2

* add: `instance_id` required input plugin setting

# v0.0.1

initial testing release
