package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	serverMetrics "github.com/kyverno/reports-server/pkg/server/metrics"
	storageMetrics "github.com/kyverno/reports-server/pkg/storage/metrics"
	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/klog/v2"
)

// Update modifies an existing resource.
//
// Semantics: STRICT - Fails if resource doesn't exist (standard Kubernetes behavior)
//
// Returns:
//   - nil on success
//   - storage.ErrNotFound if resource doesn't exist (rowsAffected=0)
//   - Other errors for database/marshaling failures
func (p *PostgresRepository[T]) Update(ctx context.Context, obj T) error {
	startTime := time.Now()
	defer func() {
		serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "update", p.resourceType, time.Since(startTime))
	}()

	// Marshal to JSON
	jsonData, err := json.Marshal(obj)
	if err != nil {
		klog.ErrorS(err, "Failed to marshal resource",
			"type", p.resourceType,
			"name", obj.GetName(),
		)
		return fmt.Errorf("failed to marshal %s: %w", p.resourceType, err)
	}

	// Build update query using query builder
	query, args := p.getQueryAndArgsForUpdate(obj, string(jsonData))

	// Execute on primary database
	db := p.router.GetWriteDB()
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		klog.ErrorS(err, "Failed to update resource",
			"type", p.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
		return fmt.Errorf("failed to update %s: %w", p.resourceType, err)
	}

	// Check if any rows were affected (strict semantics)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		klog.ErrorS(err, "Could not determine rows affected")
		return fmt.Errorf("failed to verify update: %w", err)
	}

	if rowsAffected == 0 {
		klog.V(4).InfoS("Update affected no rows (resource does not exist)",
			"type", p.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
		return storage.NewNotFoundError(p.resourceType, obj.GetName(), obj.GetNamespace())
	}

	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "update", p.resourceType)
	storageMetrics.UpdatePolicyReportMetrics("postgres", "update", obj, false)

	klog.V(4).InfoS("Updated resource",
		"type", p.resourceType,
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
	)

	return nil
}

func (p *PostgresRepository[T]) getQueryAndArgsForUpdate(obj T, jsonData string) (string, []interface{}) {
	qb := NewQueryBuilder()
	filter := storage.NewFilter(obj.GetName(), obj.GetNamespace())
	qb.ApplyFilter(filter, p.clusterID)

	query := qb.BuildUpdate(p.tableName, "report", string(jsonData))
	args := qb.Args()

	return query, args
}
