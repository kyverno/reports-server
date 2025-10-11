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

// Create inserts a new resource.
//
// Semantics: STRICT - Fails if resource already exists (standard Kubernetes behavior)
//
// Returns:
//   - nil on success
//   - storage.ErrAlreadyExists if resource already exists
//   - Other errors for database/marshaling failures
func (p *PostgresRepository[T]) Create(ctx context.Context, obj T) error {
	startTime := time.Now()
	defer func() {
		serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "create", p.resourceType, time.Since(startTime))
	}()

	// Check if resource already exists
	filter := storage.NewFilter(obj.GetName(), obj.GetNamespace())
	existing, err := p.Get(ctx, filter)
	if err == nil && existing.GetName() != "" {
		klog.V(4).InfoS("Resource already exists, cannot create",
			"type", p.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
		return storage.NewAlreadyExistsError(p.resourceType, obj.GetName(), obj.GetNamespace())
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(obj)
	if err != nil {
		klog.ErrorS(err, "Failed to marshal resource",
			"type", p.resourceType,
			"name", obj.GetName(),
		)
		return fmt.Errorf("failed to marshal %s: %w", p.resourceType, err)
	}

	// Build INSERT query (NO upsert - strict semantics)
	query, args := p.getQueryAndArgsForInsert(obj, string(jsonData))

	// Execute on primary database (writes always go to primary)
	db := p.router.GetWriteDB()
	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		klog.ErrorS(err, "Failed to create resource",
			"type", p.resourceType,
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
		return fmt.Errorf("failed to create %s: %w", p.resourceType, err)
	}

	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "create", p.resourceType)
	storageMetrics.UpdatePolicyReportMetrics("postgres", "create", obj, false)

	klog.V(4).InfoS("Created resource",
		"type", p.resourceType,
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
	)

	return nil
}

func (p *PostgresRepository[T]) getQueryAndArgsForInsert(obj T, jsonData string) (string, []interface{}) {
	qb := NewQueryBuilder()
	var columns []string
	var values []interface{}

	if p.namespaced {
		columns = []string{"name", "namespace", "report", "clusterId"}
		values = []interface{}{obj.GetName(), obj.GetNamespace(), jsonData, p.clusterID}
	} else {
		columns = []string{"name", "report", "clusterId"}
		values = []interface{}{obj.GetName(), jsonData, p.clusterID}
	}

	// upsert=false - strict CREATE semantics (fail if exists)
	query := qb.BuildInsert(p.tableName, columns, values, false)
	args := qb.Args()

	return query, args
}
