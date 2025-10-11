package storage

// Filter defines criteria for querying resources.
// All fields are optional and combined with AND logic.
type Filter struct {
	// Name filters by exact resource name (optional)
	Name string

	// Namespace filters by namespace (optional)
	// For cluster-scoped resources, this should be empty
	// For namespaced resources, empty string means "all namespaces"
	Namespace string
}

// NewFilter creates a basic filter with name and namespace.
// For more complex filters, construct Filter struct directly.
// NewFilter creates a basic filter with name and namespace.
func NewFilter(name, namespace string) Filter {
	return Filter{
		Name:      name,
		Namespace: namespace,
	}
}

// Validate checks if the filter is valid for the given operation
func (f Filter) ValidateForGet() error {
	if f.Name == "" {
		return NewInvalidFilterError("name is required for Get operation")
	}
	return nil
}

// ValidateForDelete checks if filter is valid for delete
func (f Filter) ValidateForDelete() error {
	if f.Name == "" {
		return NewInvalidFilterError("name is required for Delete operation")
	}
	return nil
}
