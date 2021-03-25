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
