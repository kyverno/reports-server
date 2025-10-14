package postgres

import (
	"fmt"
	"strings"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/klog/v2"
)

// QueryBuilder helps construct dynamic SQL queries with proper parameter binding.
// It handles PostgreSQL-style placeholders ($1, $2, etc.) and builds WHERE clauses dynamically.
type QueryBuilder struct {
	whereClauses []string
	args         []interface{}
	paramCount   int
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		whereClauses: make([]string, 0, 5),
		args:         make([]interface{}, 0, 5),
		paramCount:   1,
	}
}

// Where adds a WHERE clause with a parameter
func (qb *QueryBuilder) Where(column string, value interface{}) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, fmt.Sprintf("%s = $%d", column, qb.paramCount))
	qb.args = append(qb.args, value)
	qb.paramCount++
	return qb
}

// BuildSelect constructs a SELECT query with ORDER BY for deterministic results
func (qb *QueryBuilder) BuildSelect(tableName string, columns ...string) string {
	if len(columns) == 0 {
		columns = []string{"report"}
	}

	return fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s ORDER BY name ASC",
		strings.Join(columns, ", "),
		tableName,
		qb.WhereClause(),
	)
}

// BuildUpdate constructs an UPDATE query.
//
// The SET value is appended after WHERE parameters. This is simpler and equally correct.
//
// Typical usage:
//
//	qb := NewQueryBuilder()
//	qb.ApplyFilter(filter, clusterID)        // Adds WHERE clauses ($1, $2, $3)
//	query := qb.BuildUpdate("table", "col", value)  // Adds SET parameter ($4)
//
// Generates:
//
//	UPDATE table SET col = $4 WHERE cluster_id = $1 AND name = $2 AND namespace = $3
//	Args: [clusterID, name, namespace, value]
//
// Parameters:
//   - tableName: Table to update
//   - setColumn: Column to set (typically "report")
//   - setValue: New value for the column (typically JSON data)
//
// Returns:
//   - Complete UPDATE query with proper parameter placeholders
func (qb *QueryBuilder) BuildUpdate(tableName, setColumn string, setValue interface{}) string {
	// Append SET value after existing WHERE parameters
	setParamNumber := qb.paramCount
	qb.args = append(qb.args, setValue)
	qb.paramCount++

	return fmt.Sprintf(
		"UPDATE %s SET %s = $%d WHERE %s",
		tableName,
		setColumn,
		setParamNumber,
		qb.WhereClause(),
	)
}

// BuildDelete constructs a DELETE query.
//
// Typical usage:
//
//	qb := NewQueryBuilder()
//	qb.ApplyFilter(filter, clusterID)        // Adds WHERE clauses
//	query := qb.BuildDelete("policyreports") // Generates DELETE
//
// Generates:
//
//	DELETE FROM policyreports WHERE cluster_id = $1 AND name = $2 AND namespace = $3
//	Args: [clusterID, name, namespace]
//
// Parameters:
//   - tableName: Table to delete from
//
// Returns:
//   - Complete DELETE query with WHERE conditions
//
// Important: Always call ApplyFilter() or Where() before BuildDelete()
// to specify which resources to delete, otherwise it would delete
// everything matching "WHERE 1=1" (all rows).
func (qb *QueryBuilder) BuildDelete(tableName string) string {
	if qb.WhereClause() == "1=1" {
		klog.Warningf("Delete query without WHERE conditions for table %s", tableName)
		return ""
	}

	return fmt.Sprintf(
		"DELETE FROM %s WHERE %s",
		tableName,
		qb.WhereClause(),
	)
}

// BuildInsert constructs an INSERT query with optional upsert support.
//
// Parameters:
//   - tableName: Name of the table to insert into
//   - columns: Column names to insert (must match values length)
//   - values: Values to insert (must match columns length)
//   - upsert: If true, generates ON CONFLICT DO UPDATE clause
//
// Returns:
//   - Complete INSERT query with parameter placeholders
//
// Example:
//
//	qb := NewQueryBuilder()
//	query := qb.BuildInsert("policyreports",
//	    []string{"name", "namespace", "report", "cluster_id"},
//	    []interface{}{"my-report", "default", jsonData, "cluster-123"},
//	    true)
//
// Generates:
//
//	INSERT INTO policyreports (name, namespace, report, cluster_id)
//	VALUES ($1, $2, $3, $4)
//	ON CONFLICT (name, namespace, cluster_id)
//	DO UPDATE SET report = EXCLUDED.report
func (qb *QueryBuilder) BuildInsert(tableName string, columns []string, values []interface{}, upsert bool) string {
	// Generate placeholders for values
	placeholders := make([]string, len(values))
	for i, val := range values {
		qb.args = append(qb.args, val)
		placeholders[i] = fmt.Sprintf("$%d", qb.paramCount)
		qb.paramCount++
	}

	// Build base INSERT query
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// Add upsert logic if requested
	if upsert {
		query += qb.buildOnConflictClause(columns)
	}

	return query
}

// buildOnConflictClause generates the ON CONFLICT DO UPDATE clause.
//
// This creates PostgreSQL's upsert logic:
//
//	ON CONFLICT (primary_keys) DO UPDATE SET non_pk_col = EXCLUDED.non_pk_col
//
// Parameters:
//   - columns: All columns being inserted
//
// Returns:
//   - ON CONFLICT clause or empty string if no updateable columns
func (qb *QueryBuilder) buildOnConflictClause(columns []string) string {
	// Build UPDATE SET clauses for non-primary-key columns
	updateClauses := qb.buildUpdateClauses(columns)

	// No columns to update? No ON CONFLICT needed
	if len(updateClauses) == 0 {
		return ""
	}

	return fmt.Sprintf(
		" ON CONFLICT (%s) DO UPDATE SET %s",
		qb.buildConflictColumns(columns),
		strings.Join(updateClauses, ", "),
	)
}

// buildUpdateClauses creates "column = EXCLUDED.column" clauses for non-PK columns
func (qb *QueryBuilder) buildUpdateClauses(columns []string) []string {
	var updateClauses []string

	for _, col := range columns {
		// Skip primary key columns - they're in the conflict target
		if qb.isPrimaryKeyColumn(col) {
			continue
		}

		// For non-PK columns: column = EXCLUDED.column
		updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
	}

	return updateClauses
}

// buildConflictColumns extracts and joins primary key columns for ON CONFLICT
func (qb *QueryBuilder) buildConflictColumns(columns []string) string {
	var pkColumns []string

	for _, col := range columns {
		if qb.isPrimaryKeyColumn(col) {
			pkColumns = append(pkColumns, col)
		}
	}

	return strings.Join(pkColumns, ", ")
}

// isPrimaryKeyColumn checks if a column is part of the primary key.
// Primary keys for reports: name, namespace (if namespaced), cluster_id
func (qb *QueryBuilder) isPrimaryKeyColumn(column string) bool {
	return column == "name" || column == "namespace" || column == "cluster_id"
}

// WhereClause returns the combined WHERE clause from all conditions
func (qb *QueryBuilder) WhereClause() string {
	if len(qb.whereClauses) == 0 {
		return "1=1" // Always true if no conditions
	}
	return strings.Join(qb.whereClauses, " AND ")
}

// Args returns the accumulated query arguments in order
func (qb *QueryBuilder) Args() []interface{} {
	return qb.args
}

// ApplyFilter applies a storage.Filter to build WHERE clauses.
//
// This is the main method that converts a Filter struct into SQL WHERE conditions.
//
// Parameters:
//   - filter: Filter containing query criteria
//   - clusterID: Cluster identifier (always added)
//
// Returns:
//   - *QueryBuilder for method chaining
func (qb *QueryBuilder) ApplyFilter(filter storage.Filter, clusterID string) *QueryBuilder {
	// Always filter by cluster ID (multi-tenancy)
	qb.Where("cluster_id", clusterID)

	// Add name filter if specified
	if filter.Name != "" {
		qb.Where("name", filter.Name)
	}

	// Add namespace filter if specified
	if filter.Namespace != "" {
		qb.Where("namespace", filter.Namespace)
	}

	return qb
}
