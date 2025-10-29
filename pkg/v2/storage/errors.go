package storage

import (
	"errors"
	"fmt"
)

// Sentinel errors for storage operations.
// Use errors.Is() to check for these errors.

var (
	// ErrNotFound is returned when a resource doesn't exist
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when creating a resource that already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidFilter is returned when filter criteria are invalid or incomplete
	ErrInvalidFilter = errors.New("invalid filter")

	// ErrConnectionFailed is returned when database connection fails
	ErrConnectionFailed = errors.New("database connection failed")

	// ErrInvalidObject is returned when object validation fails
	ErrInvalidObject = errors.New("invalid object")
)

// NotFoundError wraps ErrNotFound with context
type NotFoundError struct {
	ResourceType string
	Name         string
	Namespace    string
	Err          error
}

func (e *NotFoundError) Error() string {
	if e.Namespace != "" {
		return fmt.Sprintf("%s %s/%s not found", e.ResourceType, e.Namespace, e.Name)
	}
	return fmt.Sprintf("%s %s not found", e.ResourceType, e.Name)
}

func (e *NotFoundError) Unwrap() error {
	return ErrNotFound
}

// NewNotFoundError creates a NotFoundError
func NewNotFoundError(resourceType, name, namespace string) error {
	return &NotFoundError{
		ResourceType: resourceType,
		Name:         name,
		Namespace:    namespace,
		Err:          ErrNotFound,
	}
}

// AlreadyExistsError wraps ErrAlreadyExists with context
type AlreadyExistsError struct {
	ResourceType string
	Name         string
	Namespace    string
	Err          error
}

func (e *AlreadyExistsError) Error() string {
	if e.Namespace != "" {
		return fmt.Sprintf("%s %s/%s already exists", e.ResourceType, e.Namespace, e.Name)
	}
	return fmt.Sprintf("%s %s already exists", e.ResourceType, e.Name)
}

func (e *AlreadyExistsError) Unwrap() error {
	return ErrAlreadyExists
}

// NewAlreadyExistsError creates an AlreadyExistsError
func NewAlreadyExistsError(resourceType, name, namespace string) error {
	return &AlreadyExistsError{
		ResourceType: resourceType,
		Name:         name,
		Namespace:    namespace,
		Err:          ErrAlreadyExists,
	}
}

// InvalidFilterError wraps ErrInvalidFilter with context
type InvalidFilterError struct {
	Reason string
	Err    error
}

func (e *InvalidFilterError) Error() string {
	return fmt.Sprintf("invalid filter: %s", e.Reason)
}

func (e *InvalidFilterError) Unwrap() error {
	return ErrInvalidFilter
}

// NewInvalidFilterError creates an InvalidFilterError
func NewInvalidFilterError(reason string) error {
	return &InvalidFilterError{
		Reason: reason,
		Err:    ErrInvalidFilter,
	}
}
