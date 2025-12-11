package postgres

import (
	"context"
	"database/sql"
	"sync/atomic"

	"k8s.io/klog/v2"
)

// DBRouter routes database queries between primary and read replicas.
// It provides transparent failover to primary if all replicas are unavailable.
//
// Thread-safety: This struct is safe for concurrent use. It uses atomic operations
// for the replica selection counter and does not require locks for read operations.
type DBRouter struct {
	// primary is the write database (all writes go here)
	primary *sql.DB

	// readReplicas are read-only database connections
	readReplicas []*sql.DB

	// replicaIndex is used for round-robin selection of replicas
	// Uses atomic operations to avoid locks
	replicaIndex atomic.Uint32
}

// NewDBRouter creates a new database router.
//
// Parameters:
//   - primary: The primary database connection (handles all writes)
//   - readReplicas: Optional read replica connections (can be empty)
//
// Example:
//
//	router := NewDBRouter(primaryDB, []*sql.DB{replica1, replica2})
func NewDBRouter(primary *sql.DB, readReplicas []*sql.DB) *DBRouter {
	return &DBRouter{
		primary:      primary,
		readReplicas: readReplicas,
	}
}

// GetWriteDB returns the primary database for write operations.
// All INSERT, UPDATE, DELETE operations should use this connection.
func (r *DBRouter) GetWriteDB() *sql.DB {
	return r.primary
}

// GetReadDB returns a read replica for query operations, with failover to primary.
//
// Selection strategy:
//  1. Round-robin through available read replicas
//  2. Check replica health with Ping
//  3. Fallback to primary if all replicas are unhealthy
//
// Context is used for the health check ping operation.
func (r *DBRouter) GetReadDB(ctx context.Context) *sql.DB {
	// If no replicas, use primary
	if len(r.readReplicas) == 0 {
		return r.primary
	}

	// Try replicas using round-robin
	startIdx := r.replicaIndex.Add(1) % uint32(len(r.readReplicas))

	for i := uint32(0); i < uint32(len(r.readReplicas)); i++ {
		idx := (startIdx + i) % uint32(len(r.readReplicas))
		replica := r.readReplicas[idx]

		// Quick health check - use a short timeout
		if err := replica.PingContext(ctx); err == nil {
			return replica
		}

		klog.V(5).InfoS("Read replica unavailable, trying next",
			"index", idx,
		)
	}

	// All replicas failed, fall back to primary
	klog.V(4).InfoS("All read replicas unavailable, using primary database")
	return r.primary
}

// Stats returns database connection pool statistics.
// Useful for monitoring and debugging connection pool issues.
func (r *DBRouter) Stats() DBStats {
	stats := DBStats{
		Primary: r.primary.Stats(),
	}

	stats.Replicas = make([]sql.DBStats, len(r.readReplicas))
	for i, replica := range r.readReplicas {
		stats.Replicas[i] = replica.Stats()
	}

	return stats
}

// DBStats contains statistics for all database connections.
type DBStats struct {
	Primary  sql.DBStats
	Replicas []sql.DBStats
}

// Close closes all database connections.
// Should be called during shutdown.
func (r *DBRouter) Close() error {
	// Close primary
	if err := r.primary.Close(); err != nil {
		klog.ErrorS(err, "Failed to close primary database")
		return err
	}

	// Close all replicas
	for i, replica := range r.readReplicas {
		if err := replica.Close(); err != nil {
			klog.ErrorS(err, "Failed to close read replica",
				"index", i,
			)
			// Continue closing others even if one fails
		}
	}

	return nil
}
