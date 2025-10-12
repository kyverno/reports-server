package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"k8s.io/klog/v2"
)

// TableDefinition holds information about a database table
type TableDefinition struct {
	Name       string
	Namespaced bool
}

// AllTables returns all table definitions for the reports server
func AllTables() []TableDefinition {
	return []TableDefinition{
		{"policyreports", true},
		{"clusterpolicyreports", false},
		{"ephemeralreports", true},
		{"clusterephemeralreports", false},
		{"reports", true},
		{"clusterreports", false},
	}
}

// InitializeSchema creates all necessary tables and indexes
func InitializeSchema(ctx context.Context, db *sql.DB) error {
	klog.Info("Initializing database schema")

	for _, table := range AllTables() {
		if err := createTableIfNotExists(ctx, db, table); err != nil {
			return fmt.Errorf("failed to create table %s: %w", table.Name, err)
		}

		if err := createIndexes(ctx, db, table); err != nil {
			return fmt.Errorf("failed to create indexes for table %s: %w", table.Name, err)
		}
	}

	klog.Info("Successfully initialized database schema")
	return nil
}

// createTableIfNotExists creates a table if it doesn't exist
func createTableIfNotExists(ctx context.Context, db *sql.DB, table TableDefinition) error {
	var createSQL string

	if table.Namespaced {
		createSQL = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				name VARCHAR NOT NULL,
				namespace VARCHAR NOT NULL,
				cluster_id VARCHAR NOT NULL,
				report JSONB NOT NULL,
				PRIMARY KEY(name, namespace, cluster_id)
			)
		`, table.Name)
	} else {
		createSQL = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				name VARCHAR NOT NULL,
				cluster_id VARCHAR NOT NULL,
				report JSONB NOT NULL,
				PRIMARY KEY(name, cluster_id)
			)
		`, table.Name)
	}

	_, err := db.ExecContext(ctx, createSQL)
	if err != nil {
		return fmt.Errorf("failed to execute CREATE TABLE: %w", err)
	}

	klog.V(4).InfoS("Table created or already exists", "table", table.Name)
	return nil
}

// createIndexes creates necessary indexes for a table
func createIndexes(ctx context.Context, db *sql.DB, table TableDefinition) error {
	indexes := []string{
		// Index on cluster_id for filtering by cluster (match v1 naming)
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS %scluster ON %s(cluster_id)", table.Name, table.Name),
	}

	// Add namespace index for namespaced resources
	if table.Namespaced {
		indexes = append(indexes,
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS %snamespace ON %s(namespace)", table.Name, table.Name),
		)
	}

	// Add GIN index on JSONB for label queries (optional - can be slow to create)
	// Uncomment if you need label selector support
	// indexes = append(indexes,
	// 	fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_labels ON %s USING GIN ((report->'metadata'->'labels'))", table.Name, table.Name),
	// )

	for _, indexSQL := range indexes {
		_, err := db.ExecContext(ctx, indexSQL)
		if err != nil {
			klog.ErrorS(err, "Failed to create index", "sql", indexSQL)
			// Don't fail on index creation - they're nice to have but not critical
			continue
		}
	}

	klog.V(4).InfoS("Indexes created", "table", table.Name)
	return nil
}
