package postgres

// HostConfig contains PostgreSQL host configuration for primary and read replicas.
type HostConfig struct {
	// PrimaryHost is the primary database host for write operations
	PrimaryHost string

	// ReadReplicaHosts are optional read-only replica hosts for scaling read operations
	ReadReplicaHosts []string
}

// NewHostConfig creates a new host configuration.
//
// Parameters:
//   - primaryHost: Primary database host (required, handles all writes)
//   - readReplicaHosts: Optional read replica hosts (can be nil/empty)
//
// Example:
//
//	hostConfig := NewHostConfig(
//	    "primary.db.example.com",
//	    []string{"replica1.db.example.com", "replica2.db.example.com"},
//	)
func NewHostConfig(primaryHost string, readReplicaHosts []string) *HostConfig {
	return &HostConfig{
		PrimaryHost:      primaryHost,
		ReadReplicaHosts: readReplicaHosts,
	}
}
