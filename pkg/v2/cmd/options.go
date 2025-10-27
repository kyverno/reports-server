// Package cmd provides the command-line interface for the reports-server.
// It handles flag parsing, environment variables, validation, and server initialization.
package cmd

import (
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/component-base/logs"
)

const (
	// ServerName is the name of the reports server
	ServerName = "reports-server"

	// Default values
	DefaultStorageBackend = "memory"
	DefaultDBPort         = 5432
	DefaultDBSSLMode      = "disable"
)

// Options holds all command-line options for the reports server
type Options struct {
	// Kubernetes API Server Options
	SecureServing  *genericoptions.SecureServingOptionsWithLoopback
	Authentication *genericoptions.DelegatingAuthenticationOptions
	Authorization  *genericoptions.DelegatingAuthorizationOptions
	Audit          *genericoptions.AuditOptions
	Features       *genericoptions.FeatureOptions
	Logging        *logs.Options

	// General Options
	Kubeconfig  string
	ClusterName string
	ShowVersion bool

	// Storage Options
	StorageBackend string // "postgres", "etcd", or "memory"

	// Postgres Options
	DBHost        string
	DBPort        int
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	DBSSLRootCert string
	DBSSLKey      string
	DBSSLCert     string

	// Etcd Options
	EtcdEndpoints string // Comma-separated list
	EtcdSkipTLS   bool

	// API Group Enable/Disable
	EnablePolicyReports    bool
	EnableEphemeralReports bool
	EnableOpenReports      bool

	// Service Configuration (for APIService registration)
	ServiceName      string
	ServiceNamespace string

	// Testing Options
	DisableAuthForTesting bool
}

// NewOptions creates default options
func NewOptions() *Options {
	return &Options{
		SecureServing:          genericoptions.NewSecureServingOptions().WithLoopback(),
		Authentication:         genericoptions.NewDelegatingAuthenticationOptions(),
		Authorization:          genericoptions.NewDelegatingAuthorizationOptions(),
		Audit:                  genericoptions.NewAuditOptions(),
		Features:               genericoptions.NewFeatureOptions(),
		Logging:                logs.NewOptions(),
		StorageBackend:         DefaultStorageBackend,
		DBPort:                 DefaultDBPort,
		DBSSLMode:              DefaultDBSSLMode,
		EnablePolicyReports:    true,
		EnableEphemeralReports: true,
		EnableOpenReports:      true,
		ServiceName:            ServerName,
		ServiceNamespace:       ServerName,
	}
}
