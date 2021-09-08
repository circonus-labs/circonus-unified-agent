package circonus

// special support functinos for the stackdriver_circonus input plugin ONLY...

var lookupTable = map[string]string{
	"actions.googleapis.com":              "Google Assistant Smart Home",
	"aiplatform.googleapis.com":           "AI Platform",
	"apigateway.googleapis.com":           "API Gateway",
	"apigee.googleapis.com":               "Apigee",
	"appengine.googleapis.com":            "App Engine",
	"autoscaler.googleapis.com":           "Compute Engine Autoscaler",
	"bigquery.googleapis.com":             "BigQuery",
	"bigquerybiengine.googleapis.com":     "BigQuery BI Engine",
	"bigquerydatatransfer.googleapis.com": "BigQuery Data Transfer Service",
	"bigtable.googleapis.com":             "Cloud BigTable",
	"cloudfunctions.googleapis.com":       "Cloud Functions",
	"cloudiot.googleapis.com":             "IoT Core",
	"cloudsql.googleapis.com":             "Cloud SQL",
	"cloudtasks.googleapis.com":           "Cloud Tasks",
	"cloudtrace.googleapis.com":           "Cloud Trace",
	"composer.googleapis.com":             "Cloud Composer",
	"compute.googleapis.com":              "Compute Engine",
	"dataflow.googleapis.com":             "Dataflow",
	"dataproc.googleapis.com":             "Dataproc",
	"datastore.googleapis.com":            "Datastore",
	"dlp.googleapis.com":                  "Cloud Data Loss Prevention",
	"dns.googleapis.com":                  "Cloud DNS",
	"file.googleapis.com":                 "Filestore",
	"firebaseappcheck.googleapis.com":     "Firebase (App Check)",
	"firebasedatabase.googleapis.com":     "Firebase (Database)",
	"firebasehosting.googleapis.com":      "Firebase (General)",
	"firebasestorage.googleapis.com":      "Cloud Storage for Firebase",
	"firestore.googleapis.com":            "Firestore",
	"firewallinsights.googleapis.com":     "Firewall Insights",
	"healthcare.googleapis.com":           "Cloud Healthcare API",
	"iam.googleapis.com":                  "Identity & Access Management",
	"interconnect.googleapis.com":         "Cloud Interconnect",
	"loadbalancing.googleapis.com":        "Cloud Load Balancing",
	"logging.googleapis.com":              "Cloud Logging",
	"managedidentities.googleapis.com":    "Managed Service for Microsoft Active Directory",
	"memcache.googleapis.com":             "Memorystore for Memcached",
	"metastore.googleapis.com":            "Dataproc Metastore",
	"ml.googleapis.com":                   "AI Platform",
	"monitoring.googleapis.com":           "Cloud Monitoring",
	"networking.googleapis.com":           "Network Topology",
	"networksecurity.googleapis.com":      "Google Cloud Armor",
	"privateca.googleapis.com":            "Certificate Authority Service",
	"pubsub.googleapis.com":               "Pub/Sub",
	"pubsublite.googleapis.com":           "Pub/Sub Lite",
	"recaptchaenterprise.googleapis.com":  "reCAPTCHA Enterprise",
	"recommendationengine.googleapis.com": "Recommendations AI",
	"redis.googleapis.com":                "Memorystore for Redis",
	"router.googleapis.com":               "Cloud Router",
	"run.googleapis.com":                  "Cloud Run",
	"serviceruntime.googleapis.com":       "Google Cloud APIs",
	"spanner.googleapis.com":              "Cloud Spanner",
	"storage.googleapis.com":              "Cloud Storage",
	"storagetransfer.googleapis.com":      "Storage Transfer Service (for On-Premises Data)",
	"tpu.googleapis.com":                  "Cloud TPU",
	"vpcaccess.googleapis.com":            "Virtual Private Cloud (VPC)",
	"vpn.googleapis.com":                  "Cloud VPN",
	"workflows.googleapis.com":            "Workflows",
}

// gcpMetricGroupLookup returns the "vanity" name for metric group for the check_display_name
// for stackdriver_circonus input plugin ONLY
func gcpMetricGroupLookup(metricGroup string) string {
	if vg, ok := lookupTable[metricGroup]; ok {
		return vg
	}
	return metricGroup
}

// GCPMetricTypePrefixInclude returns default list of metric type prefixes to include
// for circonus_stackdriver input plugin ONLY
func GCPMetricTypePrefixInclude() []string {
	keys := make([]string, len(lookupTable))
	i := 0
	for key := range lookupTable {
		keys[i] = key + "/"
		i++
	}
	return keys
}
