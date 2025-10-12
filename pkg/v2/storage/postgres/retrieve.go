package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

// Get retrieves a single resource by name and namespace.
// For cluster-scoped resources, pass empty string for namespace.
//
// Returns:
//   - The resource if found
//   - NotFound error if resource doesn't exist
//   - Other errors for database/marshaling failures
func (p *PostgresRepository[T]) Get(ctx context.Context, filter storage.Filter) (T, error) {
	var nilObj T

	if err := filter.ValidateForGet(); err != nil {
		return nilObj, err
	}

	query, args := p.getQueryAndArgsForSelect(filter)

	// Query from read replica (or primary as fallback)
	db := p.router.GetReadDB(ctx)
	row := db.QueryRowContext(ctx, query, args...)

	var jsonData string
	err := row.Scan(&jsonData)
	if err != nil {
		if err == sql.ErrNoRows {
			klog.V(4).InfoS("Resource not found",
				"type", p.resourceType,
				"name", filter.Name,
				"namespace", filter.Namespace,
			)
			return nilObj, errors.NewNotFound(p.gr, filter.Name)
		}
		klog.ErrorS(err, "Failed to query resource",
			"type", p.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
		)
		return nilObj, fmt.Errorf("failed to get %s %s/%s: %w", p.resourceType, filter.Namespace, filter.Name, err)
	}

	// Unmarshal JSON to typed object
	var obj T
	err = json.Unmarshal([]byte(jsonData), &obj)
	if err != nil {
		klog.ErrorS(err, "Failed to unmarshal resource",
			"type", p.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
		)
		return nilObj, fmt.Errorf("failed to unmarshal %s: %w", p.resourceType, err)
	}

	klog.V(5).InfoS("Retrieved resource",
		"type", p.resourceType,
		"name", filter.Name,
		"namespace", filter.Namespace,
	)

	return obj, nil
}

// List retrieves all resources, optionally filtered by namespace.
// For cluster-scoped resources, namespace parameter is ignored.
// Pass empty string for namespace to get all resources across all namespaces.
//
// Returns:
//   - Slice of resources
//   - Error if query fails
func (p *PostgresRepository[T]) List(ctx context.Context, filter storage.Filter) ([]T, error) {
	query, args := p.getQueryAndArgsForSelect(filter)

	// Query from read replica (or primary as fallback)
	db := p.router.GetReadDB(ctx)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		klog.ErrorS(err, "Failed to list resources",
			"type", p.resourceType,
			"namespace", filter.Namespace,
		)
		return nil, fmt.Errorf("failed to list %s: %w", p.resourceType, err)
	}
	defer rows.Close()

	// Collect results
	results := make([]T, 0, 100) // Pre-allocate reasonable capacity

	for rows.Next() {
		var jsonData string
		if err := rows.Scan(&jsonData); err != nil {
			klog.ErrorS(err, "Failed to scan row",
				"type", p.resourceType,
			)
			continue // Skip invalid rows
		}

		var obj T
		if err := json.Unmarshal([]byte(jsonData), &obj); err != nil {
			klog.ErrorS(err, "Failed to unmarshal resource",
				"type", p.resourceType,
			)
			continue // Skip invalid JSON
		}

		results = append(results, obj)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		klog.ErrorS(err, "Error iterating rows",
			"type", p.resourceType,
		)
		return nil, fmt.Errorf("error iterating %s: %w", p.resourceType, err)
	}

	klog.V(4).InfoS("Listed resources",
		"type", p.resourceType,
		"namespace", filter.Namespace,
		"count", len(results),
	)

	return results, nil
}

func (p *PostgresRepository[T]) getQueryAndArgsForSelect(filter storage.Filter) (string, []interface{}) {
	qb := NewQueryBuilder()
	qb.ApplyFilter(filter, p.clusterID)
	query := qb.BuildSelect(p.tableName)
	args := qb.Args()

	return query, args
}
