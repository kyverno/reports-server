package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/kyverno/reports-server/pkg/storage/versioning"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type genericGetter[T any, PT interface {
	*T
	metav1.Object
}] struct {
	*versioning.ResourceVersion
	typeName   string
	tableName  string
	clusterUID string
	db         *sql.DB
}

func newGenericGetter[T any, PT interface {
	*T
	metav1.Object
}](typeName, tableName, clusterUID string, db *sql.DB) *genericGetter[T, PT] {
	return &genericGetter[T, PT]{
		ResourceVersion: versioning.NewVersioning(),
		typeName:        typeName,
		tableName:       tableName,
		clusterUID:      clusterUID,
		db:              db,
	}
}

func (c *genericGetter[T, PT]) List(ctx context.Context, ns string) ([]PT, error) {
	klog.Infof("listing all %s values for namespace:%s", c.typeName, ns)
	res := make([]PT, 0)
	var jsonb string
	var rows *sql.Rows
	var err error

	if ns == "" {
		rows, err = c.db.QueryContext(ctx, fmt.Sprintf("SELECT report FROM %s WHERE cluster_id = $1", c.tableName), c.clusterUID)
	} else {
		rows, err = c.db.QueryContext(ctx, fmt.Sprintf("SELECT report FROM %s WHERE cluster_id = $1 AND namespace = $2", c.tableName), c.clusterUID, ns)
	}
	if err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to list %s", c.typeName))
		return nil, fmt.Errorf("%s list %q: %v", c.typeName, ns, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&jsonb); err != nil {
			klog.ErrorS(err, "failed to scan rows")
			return nil, fmt.Errorf("%s list %q: %v", c.typeName, ns, err)
		}
		report := PT(new(T))
		if err := json.Unmarshal([]byte(jsonb), report); err != nil {
			klog.ErrorS(err, fmt.Sprintf("failed to unmarshal %s", c.typeName))
			return nil, fmt.Errorf("%s list %q: cannot convert jsonb: %v", c.typeName, ns, err)
		}
		res = append(res, report)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (c *genericGetter[T, PT]) Get(ctx context.Context, name, ns string) (PT, error) {
	var jsonb string

	row := c.db.QueryRowContext(ctx, fmt.Sprintf("SELECT report FROM %s WHERE cluster_id = $1 AND name = $2 AND namespace = $3", c.tableName), c.clusterUID, name, ns)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("%s not found name=%s namespace=%s", c.typeName, name, ns))
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s get %s/%s: no such report", c.typeName, ns, name)
		}
		return nil, fmt.Errorf("%s get %s/%s: %v", c.typeName, ns, name, err)
	}
	report := PT(new(T))
	if err := json.Unmarshal([]byte(jsonb), report); err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to unmarshal %s", c.typeName))
		return nil, fmt.Errorf("%s get: cannot convert jsonb: %v", c.typeName, err)
	}
	return report, nil
}

func (c *genericGetter[T, PT]) Create(ctx context.Context, obj PT) error {
	if obj == nil {
		return fmt.Errorf("invalid %s", c.typeName)
	}

	name := obj.GetName()
	klog.Infof("creating %s entry for key:%s/%s", c.typeName, obj.GetNamespace(), name)
	jsonb, err := json.Marshal(obj)
	if err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to marshal %s", c.typeName))
		return err
	}

	_, err = c.db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (name, namespace, report, cluster_id) VALUES ($1, $2, $3, $4) ON CONFLICT (name, namespace, cluster_id) DO UPDATE SET report = $3", c.tableName), name, obj.GetNamespace(), string(jsonb), c.clusterUID)
	if err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to create %s", c.typeName))
		return fmt.Errorf("create %s: %v", c.typeName, err)
	}
	return nil
}

func (c *genericGetter[T, PT]) Update(ctx context.Context, obj PT) error {
	if obj == nil {
		return fmt.Errorf("invalid %s", c.typeName)
	}

	jsonb, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET report = $1 WHERE cluster_id = $2 AND namespace = $3 AND name = $4", c.tableName), string(jsonb), c.clusterUID, obj.GetNamespace(), obj.GetName())
	if err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to update %s", c.typeName))
		return fmt.Errorf("update %s: %v", c.typeName, err)
	}
	return nil
}

func (c *genericGetter[T, PT]) Delete(ctx context.Context, name, ns string) error {
	_, err := c.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE cluster_id = $1 AND namespace = $2 AND name = $3", c.tableName), c.clusterUID, ns, name)
	if err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to delete %s", c.typeName))
		return fmt.Errorf("delete %s: %v", c.typeName, err)
	}
	return nil
}

func (c *genericGetter[T, PT]) Count(ctx context.Context) (int, error) {
	var count int
	err := c.db.QueryRowContext(
		ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE cluster_id = $1", c.tableName),
		c.clusterUID,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
