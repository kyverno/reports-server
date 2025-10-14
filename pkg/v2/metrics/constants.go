package metrics

// Metric namespaces and subsystems
const (
	// Namespace for all reports-server metrics
	Namespace = "reports_server"

	// Subsystems
	SubsystemStorage = "storage"
	SubsystemAPI     = "api"
	SubsystemServer  = "server"
	SubsystemWatch   = "watch"
)

// Operation types for storage
const (
	OpCreate           = "create"
	OpGet              = "get"
	OpList             = "list"
	OpUpdate           = "update"
	OpDelete           = "delete"
	OpDeleteCollection = "delete_collection"
)

// API verbs
const (
	VerbCreate = "create"
	VerbGet    = "get"
	VerbList   = "list"
	VerbUpdate = "update"
	VerbPatch  = "patch"
	VerbDelete = "delete"
	VerbWatch  = "watch"
)

// Status codes
const (
	StatusSuccess  = "success"
	StatusError    = "error"
	StatusNotFound = "not_found"
	StatusConflict = "conflict"
	StatusInvalid  = "invalid"
)

// Resource types
const (
	ResourcePolicyReport           = "PolicyReport"
	ResourceClusterPolicyReport    = "ClusterPolicyReport"
	ResourceEphemeralReport        = "EphemeralReport"
	ResourceClusterEphemeralReport = "ClusterEphemeralReport"
	ResourceReport                 = "Report"
	ResourceClusterReport          = "ClusterReport"
)
