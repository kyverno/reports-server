package postgres

// SSLConfig contains SSL/TLS certificate configuration for PostgreSQL connections.
type SSLConfig struct {
	// Mode is the SSL mode (disable, require, verify-ca, verify-full)
	Mode string

	// RootCert is the path to the CA certificate file
	RootCert string

	// Key is the path to the client key file
	Key string

	// Cert is the path to the client certificate file
	Cert string
}

// NewSSLConfig creates a new SSL configuration.
//
// Parameters:
//   - mode: SSL mode (disable, require, verify-ca, verify-full)
//   - rootCert: Path to CA certificate (required for verify-ca/verify-full)
//   - key: Path to client key (optional)
//   - cert: Path to client certificate (optional)
//
// Example:
//
//	// For production with full verification
//	sslConfig := NewSSLConfig(
//	    "verify-full",
//	    "/etc/ssl/certs/ca.pem",
//	    "/etc/ssl/private/client-key.pem",
//	    "/etc/ssl/certs/client-cert.pem",
//	)
//
//	// For development without SSL
//	sslConfig := NewSSLConfig("disable", "", "", "")
func NewSSLConfig(mode, rootCert, key, cert string) *SSLConfig {
	return &SSLConfig{
		Mode:     mode,
		RootCert: rootCert,
		Key:      key,
		Cert:     cert,
	}
}

// IsSecure returns true if SSL is enabled (mode != "disable")
//
// Note: "require" mode enables SSL but does NOT verify the server certificate.
// For production, use "verify-full" to prevent MITM attacks.
func (s *SSLConfig) IsSecure() bool {
	return s.Mode != "disable" && s.Mode != ""
}

// IsVerified returns true if SSL certificate verification is enabled
// (verify-ca or verify-full modes)
func (s *SSLConfig) IsVerified() bool {
	return s.Mode == "verify-ca" || s.Mode == "verify-full"
}

// NewDefaultSSLConfig creates an SSL config with safe defaults for development.
// For production, use verify-full mode with proper certificates.
func NewDefaultSSLConfig() *SSLConfig {
	return &SSLConfig{
		Mode:     "disable",
		RootCert: "",
		Key:      "",
		Cert:     "",
	}
}
