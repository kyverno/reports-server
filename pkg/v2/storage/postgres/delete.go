package postgres

import (
	"context"
	"fmt"
	"time"

	serverMetrics "github.com/kyverno/reports-server/pkg/server/metrics"
	storageMetrics "github.com/kyverno/reports-server/pkg/storage/metrics"
	"github.com/kyverno/reports-server/pkg/v2/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

// Delete removes a resource matching the filter.
//
// Note: Retrieves the resource before deletion to update metrics.
// Consider adding a force flag to skip retrieval if metrics aren't needed.
//
// Returns:
//   - NotFound error if resource doesn't exist
//   - Other errors for storage failures
func (p *PostgresRepository[T]) Delete(ctx context.Context, filter storage.Filter) error {
	startTime := time.Now()
	defer func() {
		serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "delete", p.resourceType, time.Since(startTime))
	}()

	if err := filter.ValidateForDelete(); err != nil {
		return err
	}

	// Get object first for metrics (optional - can be removed for performance)
	obj, err := p.Get(ctx, filter)
	if err != nil && !errors.IsNotFound(err) {
		// Log but don't fail - deletion should proceed
		klog.V(4).InfoS("Could not retrieve resource before deletion",
			"type", p.resourceType,
			"name", filter.Name,
			"error", err,
		)
	}

	// Build delete query using query builder
	query, args := p.getQueryAndArgsForDelete(filter)

	// Execute on primary database
	db := p.router.GetWriteDB()
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		klog.ErrorS(err, "Failed to delete resource",
			"type", p.resourceType,
			"name", filter.Name,
			"namespace", filter.Namespace,
		)
		return fmt.Errorf("failed to delete %s: %w", p.resourceType, err)
	}

	// Check if any rows were deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		klog.V(4).InfoS("Could not determine rows affected", "error", err)
	} else if rowsAffected == 0 {
		klog.V(4).InfoS("Delete affected no rows (resource may not exist)",
			"type", p.resourceType,
			"name", filter.Name,
		)
		return errors.NewNotFound(p.gr, filter.Name)
	}

	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "delete", p.resourceType)

	// Update metrics if we retrieved the object
	// Check if obj is not nil by checking if GetName returns non-empty
	if err == nil && obj.GetName() != "" {
		storageMetrics.UpdatePolicyReportMetrics("postgres", "delete", obj, false)
	}

	klog.V(4).InfoS("Deleted resource",
		"type", p.resourceType,
		"name", filter.Name,
		"namespace", filter.Namespace,
	)

	return nil
}

func (p *PostgresRepository[T]) getQueryAndArgsForDelete(filter storage.Filter) (string, []interface{}) {
	qb := NewQueryBuilder()
	qb.ApplyFilter(filter, p.clusterID)
	query := qb.BuildDelete(p.tableName)
	args := qb.Args()

	return query, args
}
