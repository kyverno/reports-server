package versioning

import (
	"strconv"
	"sync"
)

// Versioning manages resource versions for optimistic concurrency control
type Versioning interface {
	// SetResourceVersion sets the resource version to the provided value if it's higher
	SetResourceVersion(version string) error

	// UseResourceVersion returns the current resource version and increments it by one
	UseResourceVersion() string
}

// Counter provides a simple counter-based versioning implementation
type Counter struct {
	mu      sync.Mutex
	version uint64
}

// NewCounter creates a new versioning counter starting at version 1
func NewCounter() *Counter {
	return &Counter{
		version: 1,
	}
}

// SetResourceVersion sets the version if the provided version is higher
func (c *Counter) SetResourceVersion(version string) error {
	v, err := strconv.ParseUint(version, 10, 64)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if v > c.version {
		c.version = v
	}

	return nil
}

// UseResourceVersion returns the current version and increments it
func (c *Counter) UseResourceVersion() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	current := c.version
	c.version++

	return strconv.FormatUint(current, 10)
}

// GetCurrentVersion returns the current version without incrementing
func (c *Counter) GetCurrentVersion() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return strconv.FormatUint(c.version, 10)
}
