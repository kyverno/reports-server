package cmd

import (
	"fmt"

	logsapi "k8s.io/component-base/logs/api/v1"
)

// Validate validates all options
func (o *Options) Validate() []error {
	var errs []error

	// Validate logging configuration
	if err := logsapi.ValidateAndApply(o.Logging, nil); err != nil {
		errs = append(errs, err)
	}

	// Validate storage backend selection
	if err := o.validateStorageBackend(); err != nil {
		errs = append(errs, err)
	}

	// Validate backend-specific options
	switch o.StorageBackend {
	case "postgres":
		if err := o.validatePostgres(); err != nil {
			errs = append(errs, err)
		}
	case "etcd":
		if err := o.validateEtcd(); err != nil {
			errs = append(errs, err)
		}
	case "memory":
		// No additional validation needed
	}

	return errs
}

// validateStorageBackend validates the storage backend selection
func (o *Options) validateStorageBackend() error {
	validBackends := map[string]bool{
		"postgres": true,
		"etcd":     true,
		"memory":   true,
	}

	if !validBackends[o.StorageBackend] {
		return fmt.Errorf("invalid storage backend %q (must be: postgres, etcd, or memory)", o.StorageBackend)
	}

	return nil
}

// validatePostgres validates PostgreSQL configuration
func (o *Options) validatePostgres() error {
	// Load from environment variables first
	o.loadPostgresFromEnv()

	// Validate required fields
	if o.DBHost == "" {
		return fmt.Errorf("postgres host is required (use --db-host flag or DB_HOST environment variable)")
	}

	if o.DBName == "" {
		return fmt.Errorf("postgres database name is required (use --db-name flag or DB_DATABASE environment variable)")
	}

	if o.DBUser == "" {
		return fmt.Errorf("postgres user is required (use --db-user flag or DB_USER environment variable)")
	}

	// Validate port range
	if o.DBPort <= 0 || o.DBPort > 65535 {
		return fmt.Errorf("invalid database port %d (must be 1-65535)", o.DBPort)
	}

	// Validate SSL mode
	validSSLModes := map[string]bool{
		"disable":     true,
		"allow":       true,
		"prefer":      true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}

	if !validSSLModes[o.DBSSLMode] {
		return fmt.Errorf("invalid SSL mode %q (must be: disable, allow, prefer, require, verify-ca, verify-full)", o.DBSSLMode)
	}

	return nil
}

// validateEtcd validates etcd configuration
func (o *Options) validateEtcd() error {
	if o.EtcdEndpoints == "" {
		return fmt.Errorf("etcd endpoints are required when using etcd storage (use --etcd-endpoints flag)")
	}

	// Parse to validate format
	endpoints := parseCSV(o.EtcdEndpoints)
	if len(endpoints) == 0 {
		return fmt.Errorf("no valid etcd endpoints found in %q", o.EtcdEndpoints)
	}

	return nil
}
