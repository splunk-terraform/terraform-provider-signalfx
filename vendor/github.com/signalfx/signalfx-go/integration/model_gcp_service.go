package integration

type GcpService string

// List of AWSService
const (
	GCP_APPENGINE        GcpService = "appengine"
	GCP_BIGQUERY         GcpService = "bigquery"
	GCP_BIGTABLE         GcpService = "bigtable"
	GCP_CLOUDFUNCTIONS   GcpService = "cloudfunctions"
	GCP_CLOUDIOT         GcpService = "cloudiot"
	GCP_CLOUDSQL         GcpService = "cloudsql"
	GCP_CLOUDTASKS       GcpService = "cloudtasks"
	GCP_COMPOSER         GcpService = "composer"
	GCP_COMPUTE          GcpService = "compute"
	GCP_CONTAINER        GcpService = "container"
	GCP_DATAFLOW         GcpService = "dataflow"
	GCP_DATAPROC         GcpService = "dataproc"
	GCP_DATASTORE        GcpService = "datastore"
	GCP_FIREBASEDATABASE GcpService = "firebasedatabase"
	GCP_FIREBASEHOSTING  GcpService = "firebasehosting"
	GCP_INTERCONNECT     GcpService = "interconnect"
	GCP_LOADBALANCING    GcpService = "loadbalancing"
	GCP_LOGGING          GcpService = "logging"
	GCP_ML               GcpService = "ml"
	GCP_MONITORING       GcpService = "monitoring"
	GCP_PUBSUB           GcpService = "pubsub"
	GCP_ROUTER           GcpService = "router"
	GCP_SERVICERUNTIME   GcpService = "serviceruntime"
	GCP_SPANNER          GcpService = "spanner"
	GCP_STORAGE          GcpService = "storage"
	GCP_VPN              GcpService = "vpn"
	GCP_FILE             GcpService = "file"
)

var GcpServiceNames = map[string]GcpService{
	"appengine":        GCP_APPENGINE,
	"bigquery":         GCP_BIGQUERY,
	"bigtable":         GCP_BIGTABLE,
	"cloudfunctions":   GCP_CLOUDFUNCTIONS,
	"cloudiot":         GCP_CLOUDIOT,
	"cloudsql":         GCP_CLOUDSQL,
	"cloudtasks":       GCP_CLOUDTASKS,
	"composer":         GCP_COMPOSER,
	"compute":          GCP_COMPUTE,
	"container":        GCP_CONTAINER,
	"dataflow":         GCP_DATAFLOW,
	"dataproc":         GCP_DATAPROC,
	"datastore":        GCP_DATASTORE,
	"firebasedatabase": GCP_FIREBASEDATABASE,
	"firebasehosting":  GCP_FIREBASEHOSTING,
	"interconnect":     GCP_INTERCONNECT,
	"loadbalancing":    GCP_LOADBALANCING,
	"logging":          GCP_LOGGING,
	"ml":               GCP_ML,
	"monitoring":       GCP_MONITORING,
	"pubsub":           GCP_PUBSUB,
	"router":           GCP_ROUTER,
	"serviceruntime":   GCP_SERVICERUNTIME,
	"spanner":          GCP_SPANNER,
	"storage":          GCP_STORAGE,
	"vpn":              GCP_VPN,
	"file":             GCP_FILE,
}
