package metrics

import (
	"sync"
)

var (
	// globalRegistry is the global metrics registry
	globalRegistry *Registry
	// once ensures global registry is initialized only once
	once sync.Once
)

// Global returns the global metrics registry
// This allows metrics to be recorded from anywhere in the codebase
func Global() *Registry {
	once.Do(func() {
		globalRegistry = NewRegistry()
	})
	return globalRegistry
}

// ResetGlobal resets the global registry (for testing)
func ResetGlobal() {
	globalRegistry = nil
	once = sync.Once{}
}
