module github.com/circonus-labs/circonus-unified-agent

go 1.16

require (
	cloud.google.com/go/kms v1.4.0 // indirect
	cloud.google.com/go/monitoring v1.6.0
	cloud.google.com/go/pubsub v1.3.1
	collectd.org v0.3.0
	github.com/Azure/azure-event-hubs-go/v3 v3.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor v0.4.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v0.3.1
	github.com/Azure/azure-storage-queue-go v0.0.0-20181215014128-6ed74e755687
	github.com/BurntSushi/toml v0.3.1
	github.com/Mellanox/rdmamap v0.0.0-20191106181932-7c3c4763a6ee
	github.com/Shopify/sarama v1.27.2
	github.com/aerospike/aerospike-client-go v1.27.0
	github.com/alecthomas/units v0.0.0-20190717042225-c3de453c63f4
	github.com/apache/thrift v0.13.0
	github.com/aristanetworks/glog v0.0.0-20191112221043-67e8567f59f3 // indirect
	github.com/aristanetworks/goarista v0.0.0-20190325233358-a123909ec740
	github.com/armon/go-metrics v0.3.0 // indirect
	github.com/aws/aws-sdk-go v1.43.21
	github.com/benbjohnson/clock v1.0.3
	github.com/bitly/go-hostpool v0.1.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869
	github.com/cenkalti/backoff v2.0.0+incompatible // indirect
	github.com/circonus-labs/go-apiclient v0.7.18
	github.com/circonus-labs/go-trapcheck v0.0.9
	github.com/circonus-labs/go-trapmetrics v0.0.10
	github.com/cisco-ie/nx-telemetry-proto v0.0.0-20220628142927-f4160bcb943c
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/couchbase/go-couchbase v0.0.0-20180501122049-16db1f1fe037
	github.com/couchbase/gomemcached v0.0.0-20180502221210-0da75df14530 // indirect
	github.com/couchbase/goutils v0.0.0-20180530154633-e865a1461c8a // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20190707035753-2be1aa521ff4
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/dimchansky/utfbom v1.1.0
	github.com/docker/docker v20.10.11+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/go-logfmt/logfmt v0.4.0
	github.com/go-ping/ping v1.1.0
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.6.0
	github.com/goburrow/modbus v0.1.0
	github.com/goburrow/serial v0.1.0 // indirect
	github.com/gobwas/glob v0.2.3
	github.com/gofrs/uuid v2.1.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/golang/geo v0.0.0-20190916061304-5b978397cfec
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.9
	github.com/google/go-github/v32 v32.1.0
	github.com/gopcua/opcua v0.1.12
	github.com/gorilla/mux v1.8.0
	github.com/gosnmp/gosnmp v1.35.0
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/harlow/kinesis-consumer v0.3.1-0.20181230152818-2f58b136fee0
	github.com/hashicorp/consul v1.2.1
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.1
	github.com/hashicorp/go-rootcerts v0.0.0-20160503143440-6bb64b370b90 // indirect
	github.com/hashicorp/go-version v1.3.0
	github.com/hashicorp/memberlist v0.1.5 // indirect
	github.com/hashicorp/serf v0.8.1 // indirect
	github.com/influxdata/go-syslog/v2 v2.0.1
	github.com/influxdata/tail v1.0.1-0.20200707181643-03a791b270e4
	github.com/influxdata/toml v0.0.0-20190415235208-270119a8ce65
	github.com/influxdata/wlog v0.0.0-20160411224016-7c63b0a71ef8
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.6.0+incompatible
	github.com/kardianos/service v1.0.0
	github.com/karrick/godirwalk v1.16.1
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/lib/pq v1.3.0 // indirect
	github.com/logzio/azure-monitor-metrics-receiver v1.0.0
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369
	github.com/mdlayher/apcupsd v0.0.0-20200608131503-2bf01da7bf1b
	github.com/mdlayher/netlink v1.4.1 // indirect
	github.com/miekg/dns v1.1.25-0.20191211073109-8ebf2e419df7
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/moby/ipvs v1.0.2
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/multiplay/go-ts3 v1.0.0
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/nats-io/nats-server/v2 v2.8.4
	github.com/nats-io/nats.go v1.16.0
	github.com/nsqio/go-nsq v1.0.8
	github.com/olivere/elastic/v7 v7.0.32
	github.com/openconfig/gnmi v0.0.0-20180912164834-33a1865c3029
	github.com/opentracing-contrib/go-observer v0.0.0-20170622124052-a52f23424492 // indirect
	github.com/openzipkin/zipkin-go-opentracing v0.3.4
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0
	github.com/prometheus/procfs v0.6.0
	github.com/riemann/riemann-go-client v0.5.0
	github.com/safchain/ethtool v0.0.0-20200218184317-f459e2d13664
	github.com/samuel/go-zookeeper v0.0.0-20180130194729-c4fab1ac1bec // indirect
	github.com/shirou/gopsutil/v3 v3.23.2
	github.com/shopspring/decimal v0.0.0-20200105231215-408a2507e114 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/streadway/amqp v0.0.0-20180528204448-e5adc2ada8b8
	github.com/stretchr/testify v1.8.2
	github.com/tbrandon/mbserver v0.0.0-20170611213546-993e1772cc62
	github.com/testcontainers/testcontainers-go v0.13.0
	github.com/tidwall/gjson v1.14.1
	github.com/valyala/fasttemplate v1.2.2
	github.com/vjeantet/grok v1.0.1
	github.com/vmware/govmomi v0.19.0
	github.com/wvanbergen/kafka v0.0.0-20171203153745-e2edea948ddf
	github.com/wvanbergen/kazoo-go v0.0.0-20180202103751-f72d8611297a // indirect
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c
	github.com/yuin/gopher-lua v0.0.0-20180630135845-46796da1b0b4 // indirect
	go.starlark.net v0.0.0-20200901195727-6e684ef5eeee
	golang.org/x/crypto v0.0.0-20220829220503-c86fa9a7ed90 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b
	golang.org/x/oauth2 v0.0.0-20220622183110-fd043fe589d2
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f
	golang.org/x/sys v0.5.0
	golang.org/x/text v0.3.7
	golang.zx2c4.com/wireguard v0.0.0-20210604143328-f9b48a961cd2 // indirect
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20210506160403-92e472f520a5
	google.golang.org/api v0.91.0
	google.golang.org/genproto v0.0.0-20220808131553-a91ffa7f803e
	google.golang.org/grpc v1.48.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/fatih/pool.v2 v2.0.0 // indirect
	gopkg.in/gorethink/gorethink.v3 v3.0.5
	gopkg.in/ldap.v3 v3.1.0
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.25.0
	k8s.io/apimachinery v0.25.0
	k8s.io/client-go v0.25.0
	modernc.org/sqlite v1.7.5
)

// replaced due to https://github.com/satori/go.uuid/issues/73
replace github.com/satori/go.uuid => github.com/gofrs/uuid v3.2.0+incompatible
