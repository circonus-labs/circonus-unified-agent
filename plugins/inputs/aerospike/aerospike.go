package aerospike

import (
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	as "github.com/aerospike/aerospike-client-go"
	"github.com/circonus-labs/circonus-unified-agent/cua"
	tlsint "github.com/circonus-labs/circonus-unified-agent/plugins/common/tls"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

type Aerospike struct {
	Servers []string `toml:"servers"`

	Username string `toml:"username"`
	Password string `toml:"password"`

	EnableTLS bool `toml:"enable_tls"`
	EnableSSL bool `toml:"enable_ssl"` // deprecated in 1.7; use enable_tls
	tlsint.ClientConfig

	initialized bool
	tlsConfig   *tls.Config

	DisableQueryNamespaces bool     `toml:"disable_query_namespaces"`
	Namespaces             []string `toml:"namespaces"`

	QuerySets bool     `toml:"query_sets"`
	Sets      []string `toml:"sets"`

	EnableTTLHistogram              bool `toml:"enable_ttl_histogram"`
	EnableObjectSizeLinearHistogram bool `toml:"enable_object_size_linear_histogram"`

	NumberHistogramBuckets int `toml:"num_histogram_buckets"`
}

var sampleConfig = `
  ## Aerospike servers to connect to (with port)
  ## This plugin will query all namespaces the aerospike
  ## server has configured and get stats for them.
  servers = ["localhost:3000"]

  # username = "circonus"
  # password = "pa$$word"

  ## Optional TLS Config
  # enable_tls = false
  # tls_ca = "/etc/circonus-unified-agent/ca.pem"
  # tls_cert = "/etc/circonus-unified-agent/cert.pem"
  # tls_key = "/etc/circonus-unified-agent/key.pem"
  ## If false, skip chain & host verification
  # insecure_skip_verify = true

  # Feature Options
  # Add namespace variable to limit the namespaces executed on
  # Leave blank to do all
  # disable_query_namespaces = true # default false
  # namespaces = ["namespace1", "namespace2"]

  # Enable set level telmetry
  # query_sets = true # default: false
  # Add namespace set combinations to limit sets executed on
  # Leave blank to do all sets
  # sets = ["namespace1/set1", "namespace1/set2", "namespace3"]

  # Histograms
  # enable_ttl_histogram = true # default: false
  # enable_object_size_linear_histogram = true # default: false

  # by default, aerospike produces a 100 bucket histogram
  # this is not great for most graphing tools, this will allow
  # the ability to squash this to a smaller number of buckets 
  # num_histogram_buckets = 100 # default: 10
`

func (a *Aerospike) SampleConfig() string {
	return sampleConfig
}

func (a *Aerospike) Description() string {
	return "Read stats from aerospike server(s)"
}

func (a *Aerospike) Gather(acc cua.Accumulator) error {
	if !a.initialized {
		tlsConfig, err := a.ClientConfig.TLSConfig()
		if err != nil {
			return fmt.Errorf("TLSConfig: %w", err)
		}
		if tlsConfig == nil && (a.EnableTLS || a.EnableSSL) {
			tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		a.tlsConfig = tlsConfig
		a.initialized = true
	}

	switch {
	case a.NumberHistogramBuckets == 0:
		a.NumberHistogramBuckets = 10
	case a.NumberHistogramBuckets > 100:
		a.NumberHistogramBuckets = 100
	case a.NumberHistogramBuckets < 1:
		a.NumberHistogramBuckets = 10
	}

	if len(a.Servers) == 0 {
		return a.gatherServer("127.0.0.1:3000", acc)
	}

	var wg sync.WaitGroup
	wg.Add(len(a.Servers))
	for _, server := range a.Servers {
		go func(serv string) {
			defer wg.Done()
			acc.AddError(a.gatherServer(serv, acc))
		}(server)
	}

	wg.Wait()
	return nil
}

func (a *Aerospike) gatherServer(hostPort string, acc cua.Accumulator) error {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return fmt.Errorf("split host (%s): %w", hostPort, err)
	}

	iport, err := strconv.Atoi(port)
	if err != nil {
		iport = 3000
	}

	policy := as.NewClientPolicy()
	policy.User = a.Username
	policy.Password = a.Password
	policy.TlsConfig = a.tlsConfig
	c, err := as.NewClientWithPolicy(policy, host, iport)
	if err != nil {
		return fmt.Errorf("new client with policy: %w", err)
	}
	defer c.Close()

	nodes := c.GetNodes()
	for _, n := range nodes {
		stats, err := a.getNodeInfo(n)
		if err != nil {
			return err
		}
		a.parseNodeInfo(stats, hostPort, n.GetName(), acc)

		namespaces, err := a.getNamespaces(n)
		if err != nil {
			return err
		}

		if !a.DisableQueryNamespaces {
			// Query Namespaces
			for _, namespace := range namespaces {
				stats, err = a.getNamespaceInfo(namespace, n)

				if err != nil {
					continue
				} else {
					a.parseNamespaceInfo(stats, hostPort, namespace, n.GetName(), acc)
				}

				if a.EnableTTLHistogram {
					err = a.getTTLHistogram(hostPort, namespace, "", n, acc)
					if err != nil {
						continue
					}
				}
				if a.EnableObjectSizeLinearHistogram {
					err = a.getObjectSizeLinearHistogram(hostPort, namespace, "", n, acc)
					if err != nil {
						continue
					}
				}
			}
		}

		if a.QuerySets {
			namespaceSets, err := a.getSets(n)
			if err == nil {
				for _, namespaceSet := range namespaceSets {
					namespace, set := splitNamespaceSet(namespaceSet)

					stats, err := a.getSetInfo(namespaceSet, n)

					if err != nil {
						continue
					} else {
						a.parseSetInfo(stats, hostPort, namespaceSet, n.GetName(), acc)
					}

					if a.EnableTTLHistogram {
						err = a.getTTLHistogram(hostPort, namespace, set, n, acc)
						if err != nil {
							continue
						}
					}

					if a.EnableObjectSizeLinearHistogram {
						err = a.getObjectSizeLinearHistogram(hostPort, namespace, set, n, acc)
						if err != nil {
							continue
						}
					}
				}
			}
		}
	}
	return nil
}

func (a *Aerospike) getNodeInfo(n *as.Node) (map[string]string, error) {
	stats, err := as.RequestNodeStats(n)
	if err != nil {
		return nil, fmt.Errorf("request node stats: %w", err)
	}

	return stats, nil
}

func (a *Aerospike) parseNodeInfo(stats map[string]string, hostPort string, nodeName string, acc cua.Accumulator) {
	tags := map[string]string{
		"aerospike_host": hostPort,
		"node_name":      nodeName,
	}
	fields := make(map[string]interface{})

	for k, v := range stats {
		val := parseValue(v)
		fields[strings.ReplaceAll(k, "-", "_")] = val
	}
	acc.AddFields("aerospike_node", fields, tags, time.Now())
}

func (a *Aerospike) getNamespaces(n *as.Node) ([]string, error) {
	var namespaces []string
	if len(a.Namespaces) == 0 {
		info, err := as.RequestNodeInfo(n, "namespaces")
		if err != nil {
			return namespaces, fmt.Errorf("request node info: %w", err)
		}
		namespaces = strings.Split(info["namespaces"], ";")
	} else {
		namespaces = a.Namespaces
	}

	return namespaces, nil
}

func (a *Aerospike) getNamespaceInfo(namespace string, n *as.Node) (map[string]string, error) {
	stats, err := as.RequestNodeInfo(n, "namespace/"+namespace)
	if err != nil {
		return nil, fmt.Errorf("request node info: %w", err)
	}

	return stats, nil
}
func (a *Aerospike) parseNamespaceInfo(stats map[string]string, hostPort string, namespace string, nodeName string, acc cua.Accumulator) {

	nTags := map[string]string{
		"aerospike_host": hostPort,
		"node_name":      nodeName,
	}
	nTags["namespace"] = namespace
	nFields := make(map[string]interface{})

	stat := strings.Split(stats["namespace/"+namespace], ";")
	for _, pair := range stat {
		parts := strings.Split(pair, "=")
		if len(parts) < 2 {
			continue
		}
		val := parseValue(parts[1])
		nFields[strings.ReplaceAll(parts[0], "-", "_")] = val
	}
	acc.AddFields("aerospike_namespace", nFields, nTags, time.Now())
}

func (a *Aerospike) getSets(n *as.Node) ([]string, error) {
	var namespaceSets []string
	// Gather all sets
	if len(a.Sets) == 0 {
		stats, err := as.RequestNodeInfo(n, "sets")
		if err != nil {
			return namespaceSets, fmt.Errorf("request node info: %w", err)
		}

		stat := strings.Split(stats["sets"], ";")
		for _, setStats := range stat {
			// setInfo is "ns=test:set=foo:objects=1:tombstones=0"
			if len(setStats) > 0 {
				pairs := strings.Split(setStats, ":")
				var ns, set string
				for _, pair := range pairs {
					parts := strings.Split(pair, "=")
					if len(parts) == 2 {
						if parts[0] == "ns" {
							ns = parts[1]
						}
						if parts[0] == "set" {
							set = parts[1]
						}
					}
				}
				if len(ns) > 0 && len(set) > 0 {
					namespaceSets = append(namespaceSets, fmt.Sprintf("%s/%s", ns, set))
				}
			}
		}
	} else { // User has passed in sets
		namespaceSets = a.Sets
	}

	return namespaceSets, nil
}

func (a *Aerospike) getSetInfo(namespaceSet string, n *as.Node) (map[string]string, error) {
	stats, err := as.RequestNodeInfo(n, "sets/"+namespaceSet)
	if err != nil {
		return nil, fmt.Errorf("request node info: %w", err)
	}
	return stats, nil
}

func (a *Aerospike) parseSetInfo(stats map[string]string, hostPort string, namespaceSet string, nodeName string, acc cua.Accumulator) {

	stat := strings.Split(
		strings.TrimSuffix(
			stats[fmt.Sprintf("sets/%s", namespaceSet)], ";"), ":")
	nTags := map[string]string{
		"aerospike_host": hostPort,
		"node_name":      nodeName,
		"set":            namespaceSet,
	}
	nFields := make(map[string]interface{})
	for _, part := range stat {
		pieces := strings.Split(part, "=")
		if len(pieces) < 2 {
			continue
		}

		val := parseValue(pieces[1])
		nFields[strings.ReplaceAll(pieces[0], "-", "_")] = val
	}
	acc.AddFields("aerospike_set", nFields, nTags, time.Now())
}

func (a *Aerospike) getTTLHistogram(hostPort string, namespace string, set string, n *as.Node, acc cua.Accumulator) error {
	stats, err := a.getHistogram(namespace, set, "ttl", n)
	if err != nil {
		return err
	}
	a.parseHistogram(stats, hostPort, namespace, set, "ttl", n.GetName(), acc)

	return nil
}

func (a *Aerospike) getObjectSizeLinearHistogram(hostPort string, namespace string, set string, n *as.Node, acc cua.Accumulator) error {

	stats, err := a.getHistogram(namespace, set, "object-size-linear", n)
	if err != nil {
		return err
	}
	a.parseHistogram(stats, hostPort, namespace, set, "object-size-linear", n.GetName(), acc)

	return nil
}

func (a *Aerospike) getHistogram(namespace string, set string, histogramType string, n *as.Node) (map[string]string, error) {
	var queryArg string
	if len(set) > 0 {
		queryArg = fmt.Sprintf("histogram:type=%s;namespace=%v;set=%v", histogramType, namespace, set)
	} else {
		queryArg = fmt.Sprintf("histogram:type=%s;namespace=%v", histogramType, namespace)
	}

	stats, err := as.RequestNodeInfo(n, queryArg)
	if err != nil {
		return nil, fmt.Errorf("request node info: %w", err)
	}
	return stats, nil

}

func (a *Aerospike) parseHistogram(stats map[string]string, hostPort string, namespace string, set string, histogramType string, nodeName string, acc cua.Accumulator) {

	nTags := map[string]string{
		"aerospike_host": hostPort,
		"node_name":      nodeName,
		"namespace":      namespace,
	}

	if len(set) > 0 {
		nTags["set"] = set
	}

	nFields := make(map[string]interface{})

	for _, stat := range stats {
		for _, part := range strings.Split(stat, ":") {
			pieces := strings.Split(part, "=")
			if len(pieces) < 2 {
				continue
			}

			if pieces[0] == "buckets" {
				buckets := strings.Split(pieces[1], ",")

				// Normalize incase of less buckets than expected
				numRecordsPerBucket := 1
				if len(buckets) > a.NumberHistogramBuckets {
					numRecordsPerBucket = int(math.Ceil((float64(len(buckets)) / float64(a.NumberHistogramBuckets))))
				}

				bucketCount := 0
				bucketSum := int64(0) // cast to int64, as can have large object sums
				bucketName := 0
				for i, bucket := range buckets {
					// Sum records and increment bucket collection counter
					if bucketCount < numRecordsPerBucket {
						bucketSum += parseValue(bucket).(int64)
						bucketCount++
					}

					// Store records and reset counters
					// increment bucket name
					if bucketCount == numRecordsPerBucket {
						nFields[strconv.Itoa(bucketName)] = bucketSum

						bucketCount = 0
						bucketSum = 0
						bucketName++
					} else if i == (len(buckets) - 1) {
						// base/edge case where final bucket does not fully
						// fill number of records per bucket
						nFields[strconv.Itoa(bucketName)] = bucketSum
					}
				}

			}
		}
	}

	acc.AddFields(fmt.Sprintf("aerospike_histogram_%v", strings.ReplaceAll(histogramType, "-", "_")), nFields, nTags, time.Now())
}

func splitNamespaceSet(namespaceSet string) (string, string) {
	split := strings.Split(namespaceSet, "/")
	return split[0], split[1]
}

func parseValue(v string) interface{} {
	if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
		return parsed
	} else if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
		return parsed
	} else if parsed, err := strconv.ParseBool(v); err == nil {
		return parsed
	} else {
		// leave as string
		return v
	}
}

// func copyTags(m map[string]string) map[string]string {
// 	out := make(map[string]string)
// 	for k, v := range m {
// 		out[k] = v
// 	}
// 	return out
// }

func init() {
	inputs.Add("aerospike", func() cua.Input {
		return &Aerospike{}
	})
}
