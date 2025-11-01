package postgres

import "fmt"

// Config holds PostgreSQL database configuration for reports-server.
// It supports primary database and optional read replicas for scaling read operations.
type Config struct {
	// HostConfig contains primary and replica host information
	HostConfig *HostConfig

	// Port is the PostgreSQL port (typically 5432)
	Port int

	// User is the database username
	User string

	// Password is the database password
	Password string

	// DBname is the database name
	DBname string

	// SSLConfig contains SSL/TLS certificate configuration
	SSLConfig *SSLConfig
}

// NewConfig creates a new PostgreSQL configuration.
//
// Parameters:
//   - hostConfig: Primary and replica host configuration
//   - port: PostgreSQL port (typically 5432)
//   - user: Database username
//   - password: Database password
//   - dbname: Database name
//   - sslConfig: SSL/TLS configuration
//
// Returns:
//   - *Config: Configuration instance
func NewConfig(
	hostConfig *HostConfig,
	port int,
	user, password, dbname string,
	sslConfig *SSLConfig,
) *Config {
	return &Config{
		HostConfig: hostConfig,
		Port:       port,
		User:       user,
		Password:   password,
		DBname:     dbname,
		SSLConfig:  sslConfig,
	}
}

// PrimaryDSN generates a PostgreSQL connection string (DSN) for the PRIMARY database.
//
// WARNING: This returns credentials in plaintext. Never log this directly.
// Use SafeString() for logging purposes.
//
// Note: This returns the DSN for the primary database only.
// Read replicas must be connected separately using ReadReplicaDSNs().
//
// Format: "host=X port=Y user=Z password=W dbname=D sslmode=M sslrootcert=R sslkey=K sslcert=C"
//
// Returns:
//   - Complete PostgreSQL DSN with credentials
func (c *Config) PrimaryDSN() string {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.HostConfig.PrimaryHost, c.Port, c.User, c.Password, c.DBname, c.SSLConfig.Mode)

	if c.SSLConfig.RootCert != "" {
		connStr += fmt.Sprintf(" sslrootcert=%s", c.SSLConfig.RootCert)
	}
	if c.SSLConfig.Key != "" {
		connStr += fmt.Sprintf(" sslkey=%s", c.SSLConfig.Key)
	}
	if c.SSLConfig.Cert != "" {
		connStr += fmt.Sprintf(" sslcert=%s", c.SSLConfig.Cert)
	}

	return connStr
}

// SafeString returns a connection string with password redacted for safe logging.
//
// Example output: "host=localhost port=5432 user=reports password=*** dbname=reportsdb sslmode=require"
//
// Use this for logging, debugging, or displaying configuration.
func (c *Config) SafeString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=*** dbname=%s sslmode=%s",
		c.HostConfig.PrimaryHost, c.Port, c.User, c.DBname, c.SSLConfig.Mode)
}

// ReadReplicaDSNs generates PostgreSQL DSNs for all read replica hosts.
//
// WARNING: These contain credentials in plaintext. Never log directly.
//
// Note: All replicas use the same credentials and SSL config as primary.
// Each replica DSN includes full SSL certificate paths if configured.
//
// Returns:
//   - Slice of complete PostgreSQL DSNs for read replicas
func (c *Config) ReadReplicaDSNs() []string {
	replicaStrings := make([]string, len(c.HostConfig.ReadReplicaHosts))
	for i, host := range c.HostConfig.ReadReplicaHosts {
		connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			host, c.Port, c.User, c.Password, c.DBname, c.SSLConfig.Mode)

		// Add SSL certificates if configured (same as primary)
		if c.SSLConfig.RootCert != "" {
			connStr += fmt.Sprintf(" sslrootcert=%s", c.SSLConfig.RootCert)
		}
		if c.SSLConfig.Key != "" {
			connStr += fmt.Sprintf(" sslkey=%s", c.SSLConfig.Key)
		}
		if c.SSLConfig.Cert != "" {
			connStr += fmt.Sprintf(" sslcert=%s", c.SSLConfig.Cert)
		}

		replicaStrings[i] = connStr
	}

	return replicaStrings
}

// Validate checks if the configuration is valid.
//
// Returns:
//   - error: Validation error if configuration is invalid
func (c *Config) Validate() error {
	if c.HostConfig == nil {
		return fmt.Errorf("host configuration is required")
	}

	if c.HostConfig.PrimaryHost == "" {
		return fmt.Errorf("primary host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Port)
	}

	if c.User == "" {
		return fmt.Errorf("database user is required")
	}

	// Password validation: Empty password is allowed for peer auth or trust mode,
	// but warn if using password-based auth without a password
	if c.Password == "" && c.SSLConfig != nil && c.SSLConfig.Mode != "disable" {
		// This is just a warning case - some setups use cert-based auth
		// Don't fail, but log it
	}

	if c.DBname == "" {
		return fmt.Errorf("database name is required")
	}

	if c.SSLConfig == nil {
		return fmt.Errorf("SSL configuration is required (use NewSSLConfig(\"disable\", \"\", \"\", \"\") for no SSL)")
	}

	// Validate SSL mode is one of the allowed values
	validSSLModes := map[string]bool{
		"disable":     true,
		"allow":       true,
		"prefer":      true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}

	if !validSSLModes[c.SSLConfig.Mode] {
		return fmt.Errorf("invalid SSL mode: %q (must be one of: disable, allow, prefer, require, verify-ca, verify-full)", c.SSLConfig.Mode)
	}

	// Validate SSL certificate requirements
	if c.SSLConfig.Mode == "verify-ca" || c.SSLConfig.Mode == "verify-full" {
		if c.SSLConfig.RootCert == "" {
			return fmt.Errorf("SSL mode %q requires sslrootcert to be set", c.SSLConfig.Mode)
		}
	}

	return nil
}

// GetPrimaryHost is a convenience method to get the primary host.
func (c *Config) GetPrimaryHost() string {
	return c.HostConfig.PrimaryHost
}

// GetReadReplicaHosts is a convenience method to get read replica hosts.
func (c *Config) GetReadReplicaHosts() []string {
	return c.HostConfig.ReadReplicaHosts
}
