package all

//nolint:golint
import (
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/activemq"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/aerospike"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/amqp_consumer"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/apache"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/apcupsd"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/aurora"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/azure_monitor"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/azure_storage_queue"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/bcache"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/beanstalkd"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/bind"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/bond"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/burrow"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/ceph"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/cgroup"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/chrony"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/circ_http_json"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/cisco_telemetry_mdt"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/clickhouse"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/cloud_pubsub"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/cloud_pubsub_push"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/cloudwatch"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/conntrack"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/consul"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/couchbase"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/couchdb"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/cpu"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/dcos"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/disk"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/diskio"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/disque"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/dmcache"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/dns_query"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/docker"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/docker_log"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/dovecot"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/ecs"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/elasticsearch"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/ethtool"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/eventhub_consumer"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/exec"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/execd"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/fail2ban"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/fibaro"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/file"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/filecount"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/filestat"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/fireboard"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/fluentd"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/github"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/gnmi"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/graylog"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/haproxy"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/hddtemp"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/http"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/http_listener_v2"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/http_response"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/icinga2"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/infiniband"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/influxdb"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/influxdb_listener"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/influxdb_v2_listener"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/intel_rdt"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/internal"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/interrupts"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/ipmi_sensor"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/ipset"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/iptables"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/ipvs"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/jenkins"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/jolokia2"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/jti_openconfig_telemetry"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/kafka_consumer"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/kafka_consumer_legacy"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/kapacitor"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/kernel"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/kernel_vmstat"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/kibana"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/kinesis_consumer"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/kube_inventory"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/kubernetes"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/lanz"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/leofs"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/linux_sysctl_fs"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/logstash"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/lustre2"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/mailchimp"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/marklogic"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/mcrouter"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/mem"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/memcached"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/mesos"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/minecraft"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/modbus"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/mongodb"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/monit"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/mqtt_consumer"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/multifile"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/mysql"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nats"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nats_consumer"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/neptune_apex"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/net"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/net_response"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nginx"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nginx_plus"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nginx_plus_api"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nginx_sts"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nginx_upstream_check"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nginx_vts"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nsd"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nsq"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nsq_consumer"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nstat"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/ntpq"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/nvidia_smi"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/opcua"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/openldap"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/openntpd"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/opensmtpd"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/openweathermap"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/passenger"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/pf"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/pgbouncer"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/phpfpm"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/ping"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/postfix"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/postgresql"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/postgresql_extensible"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/powerdns"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/powerdns_recursor"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/processes"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/procstat"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/prometheus"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/proxmox"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/puppetagent"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/rabbitmq"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/raindrops"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/redfish"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/redis"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/rethinkdb"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/riak"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/salesforce"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/sensors"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/sflow"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/smart"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/snmp"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/snmp_trap"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/socket_listener"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/solr"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/sqlserver"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/stackdriver"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/stackdriver_circonus"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/statsd"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/suricata"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/swap"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/synproxy"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/syslog"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/sysstat"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/system"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/systemd_units"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/tail"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/teamspeak"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/temp"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/tengine"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/tomcat"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/trig"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/twemproxy"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/unbound"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/uwsgi"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/varnish"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/vsphere"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/webhooks"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/win_eventlog"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/win_perf_counters"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/win_services"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/wireguard"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/wireless"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/x509_cert"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/zfs"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/zipkin"
	_ "github.com/circonus-labs/circonus-unified-agent/plugins/inputs/zookeeper"
)
