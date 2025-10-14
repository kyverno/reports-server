package logging

// Log levels used throughout the v2 codebase
// Based on klog verbosity levels
const (
	// LevelError (0) - Always logged, for errors only
	// Use: klog.ErrorS(err, "message")
	LevelError = 0

	// LevelWarning (1) - Warnings and important events
	// Use: klog.V(1).InfoS("message")
	LevelWarning = 1

	// LevelInfo (2) - General information about operations
	// Use: klog.V(2).InfoS("message")
	// Examples: Server starting, API group installed, shutdown
	LevelInfo = 2

	// LevelDebug (4) - Detailed debugging information
	// Use: klog.V(4).InfoS("message")
	// Examples: Individual CRUD operations, filtering results
	LevelDebug = 4

	// LevelTrace (6) - Very detailed trace information
	// Use: klog.V(6).InfoS("message")
	// Examples: Every function call, parameter values
	LevelTrace = 6
)

// Guidelines:
//
// Use klog.ErrorS() for errors (always logged):
//   klog.ErrorS(err, "Failed to create resource", "kind", "PolicyReport", "name", name)
//
// Use klog.InfoS() for important events (always logged):
//   klog.InfoS("Server starting", "version", version)
//
// Use klog.V(2).InfoS() for general info:
//   klog.V(2).InfoS("API group installed", "group", "wgpolicyk8s.io")
//
// Use klog.V(4).InfoS() for debug/operation details:
//   klog.V(4).InfoS("Creating resource", "kind", kind, "name", name, "namespace", ns)
//
// Use klog.V(6).InfoS() for trace/verbose debugging:
//   klog.V(6).InfoS("Filtering items", "total", total, "matched", matched, "labels", labels)
