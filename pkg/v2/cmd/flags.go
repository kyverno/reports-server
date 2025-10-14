package cmd

import (
	cliflag "k8s.io/component-base/cli/flag"
	logsapi "k8s.io/component-base/logs/api/v1"
)

// Flags returns all command-line flags organized by category
func (o *Options) Flags() cliflag.NamedFlagSets {
	fs := cliflag.NamedFlagSets{}

	o.addGeneralFlags(&fs)
	o.addStorageFlags(&fs)
	o.addPostgresFlags(&fs)
	o.addEtcdFlags(&fs)
	o.addAPIGroupFlags(&fs)
	o.addServiceFlags(&fs)
	o.addKubernetesFlags(&fs)

	return fs
}

// addGeneralFlags adds general command flags
func (o *Options) addGeneralFlags(fs *cliflag.NamedFlagSets) {
	generalFS := fs.FlagSet("general")
	generalFS.BoolVar(&o.ShowVersion, "version", false,
		"Show version and exit")
	generalFS.StringVar(&o.Kubeconfig, "kubeconfig", "",
		"Path to kubeconfig file (defaults to in-cluster config)")
	generalFS.StringVar(&o.ClusterName, "cluster-name", "",
		"Human-readable cluster name for identification")
}

// addStorageFlags adds storage backend selection flags
func (o *Options) addStorageFlags(fs *cliflag.NamedFlagSets) {
	storageFS := fs.FlagSet("storage")
	storageFS.StringVar(&o.StorageBackend, "storage-backend", DefaultStorageBackend,
		"Storage backend to use: postgres, etcd, or memory")
}

// addPostgresFlags adds PostgreSQL-specific flags
func (o *Options) addPostgresFlags(fs *cliflag.NamedFlagSets) {
	postgresFS := fs.FlagSet("postgres")

	postgresFS.StringVar(&o.DBHost, "db-host", "",
		"PostgreSQL host address (can also use DB_HOST env var)")
	postgresFS.IntVar(&o.DBPort, "db-port", DefaultDBPort,
		"PostgreSQL port (can also use DB_PORT env var)")
	postgresFS.StringVar(&o.DBUser, "db-user", "",
		"PostgreSQL username (can also use DB_USER env var)")
	postgresFS.StringVar(&o.DBPassword, "db-password", "",
		"PostgreSQL password (recommended: use DB_PASSWORD env var)")
	postgresFS.StringVar(&o.DBName, "db-name", "",
		"PostgreSQL database name (can also use DB_DATABASE env var)")

	postgresFS.StringVar(&o.DBSSLMode, "db-ssl-mode", DefaultDBSSLMode,
		"PostgreSQL SSL mode: disable, require, verify-ca, or verify-full")
	postgresFS.StringVar(&o.DBSSLRootCert, "db-ssl-rootcert", "",
		"Path to PostgreSQL SSL root certificate")
	postgresFS.StringVar(&o.DBSSLKey, "db-ssl-key", "",
		"Path to PostgreSQL SSL client key")
	postgresFS.StringVar(&o.DBSSLCert, "db-ssl-cert", "",
		"Path to PostgreSQL SSL client certificate")
}

// addEtcdFlags adds etcd-specific flags
func (o *Options) addEtcdFlags(fs *cliflag.NamedFlagSets) {
	etcdFS := fs.FlagSet("etcd")

	etcdFS.StringVar(&o.EtcdEndpoints, "etcd-endpoints", "",
		"Etcd endpoints (comma-separated, e.g., localhost:2379,localhost:2380)")
	etcdFS.BoolVar(&o.EtcdSkipTLS, "etcd-skip-tls", true,
		"Skip TLS verification when connecting to etcd")
}

// addAPIGroupFlags adds API group enable/disable flags
func (o *Options) addAPIGroupFlags(fs *cliflag.NamedFlagSets) {
	apiFS := fs.FlagSet("api-groups")

	apiFS.BoolVar(&o.EnablePolicyReports, "enable-policy-reports", true,
		"Enable wgpolicyk8s.io/v1alpha2 API group (PolicyReport, ClusterPolicyReport)")
	apiFS.BoolVar(&o.EnableEphemeralReports, "enable-ephemeral-reports", true,
		"Enable reports.kyverno.io/v1 API group (EphemeralReport, ClusterEphemeralReport)")
	apiFS.BoolVar(&o.EnableOpenReports, "enable-open-reports", true,
		"Enable openreports.io/v1alpha1 API group (Report, ClusterReport)")
}

// addServiceFlags adds APIService configuration flags
func (o *Options) addServiceFlags(fs *cliflag.NamedFlagSets) {
	serviceFS := fs.FlagSet("apiservice")

	serviceFS.StringVar(&o.ServiceName, "service-name", ServerName,
		"Service name for APIService registration")
	serviceFS.StringVar(&o.ServiceNamespace, "service-namespace", ServerName,
		"Service namespace for APIService registration")
}

// addKubernetesFlags adds Kubernetes API server flags
func (o *Options) addKubernetesFlags(fs *cliflag.NamedFlagSets) {
	o.SecureServing.AddFlags(fs.FlagSet("secure-serving"))
	o.Authentication.AddFlags(fs.FlagSet("authentication"))
	o.Authorization.AddFlags(fs.FlagSet("authorization"))
	o.Audit.AddFlags(fs.FlagSet("audit"))
	o.Features.AddFlags(fs.FlagSet("features"))
	logsapi.AddFlags(o.Logging, fs.FlagSet("logging"))
}
