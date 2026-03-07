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

type genericClusterGetter[T any, PT interface {
	*T
	metav1.Object
}] struct {
	*versioning.ResourceVersion
	typeName   string
	tableName  string
	clusterUID string
	db         *sql.DB
}

func newGenericClusterGetter[T any, PT interface {
	*T
	metav1.Object
}](typeName, tableName, clusterUID string, db *sql.DB) *genericClusterGetter[T, PT] {
	return &genericClusterGetter[T, PT]{
		ResourceVersion: versioning.NewVersioning(),
		typeName:        typeName,
		tableName:       tableName,
		clusterUID:      clusterUID,
		db:              db,
	}
}

func (c *genericClusterGetter[T, PT]) List(ctx context.Context) ([]PT, error) {
	klog.Infof("listing all %s values", c.typeName)
	res := make([]PT, 0)
	var jsonb string

	rows, err := c.db.Query(fmt.Sprintf("SELECT report FROM %s WHERE cluster_id = $1", c.tableName), c.clusterUID)
	if err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to list %s", c.typeName))
		return nil, fmt.Errorf("%s list: %v", c.typeName, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&jsonb); err != nil {
			klog.ErrorS(err, "failed to scan rows")
			return nil, fmt.Errorf("%s list: %v", c.typeName, err)
		}
		report := PT(new(T))
		if err := json.Unmarshal([]byte(jsonb), report); err != nil {
			klog.ErrorS(err, fmt.Sprintf("failed to unmarshal %s", c.typeName))
			return nil, fmt.Errorf("%s list: cannot convert jsonb: %v", c.typeName, err)
		}
		res = append(res, report)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (c *genericClusterGetter[T, PT]) Get(ctx context.Context, name string) (PT, error) {
	var jsonb string

	row := c.db.QueryRow(fmt.Sprintf("SELECT report FROM %s WHERE cluster_id = $1 AND name = $2", c.tableName), c.clusterUID, name)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("%s not found name=%s", c.typeName, name))
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s get %s: no such report", c.typeName, name)
		}
		return nil, fmt.Errorf("%s get %s: %v", c.typeName, name, err)
	}
	report := PT(new(T))
	if err := json.Unmarshal([]byte(jsonb), report); err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to unmarshal %s", c.typeName))
		return nil, fmt.Errorf("%s get: cannot convert jsonb: %v", c.typeName, err)
	}
	return report, nil
}

func (c *genericClusterGetter[T, PT]) Create(ctx context.Context, obj PT) error {
	if obj == nil {
		return fmt.Errorf("invalid %s", c.typeName)
	}

	name := obj.GetName()
	klog.Infof("creating %s entry for key:%s", c.typeName, name)
	jsonb, err := json.Marshal(obj)
	if err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to marshal %s", c.typeName))
		return err
	}

	_, err = c.db.Exec(fmt.Sprintf("INSERT INTO %s (name, report, cluster_id) VALUES ($1, $2, $3) ON CONFLICT (name, cluster_id) DO UPDATE SET report = $2", c.tableName), name, string(jsonb), c.clusterUID)
	if err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to create %s", c.typeName))
		return fmt.Errorf("create %s: %v", c.typeName, err)
	}
	return nil
}

func (c *genericClusterGetter[T, PT]) Update(ctx context.Context, obj PT) error {
	if obj == nil {
		return fmt.Errorf("invalid %s", c.typeName)
	}

	jsonb, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = c.db.Exec(fmt.Sprintf("UPDATE %s SET report = $1 WHERE cluster_id = $2 AND name = $3", c.tableName), string(jsonb), c.clusterUID, obj.GetName())
	if err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to update %s", c.typeName))
		return fmt.Errorf("update %s: %v", c.typeName, err)
	}
	return nil
}

func (c *genericClusterGetter[T, PT]) Delete(ctx context.Context, name string) error {
	_, err := c.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE cluster_id = $1 AND name = $2", c.tableName), c.clusterUID, name)
	if err != nil {
		klog.ErrorS(err, fmt.Sprintf("failed to delete %s", c.typeName))
		return fmt.Errorf("delete %s: %v", c.typeName, err)
	}
	return nil
}
