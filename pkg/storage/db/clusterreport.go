package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"k8s.io/klog/v2"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
)

type orClusterReportDB struct {
	sync.Mutex
	db         *sql.DB
	clusterUID string
}

func (o *orClusterReportDB) List(ctx context.Context) ([]*openreportsv1alpha1.ClusterReport, error) {
	o.Lock()
	defer o.Unlock()

	klog.Infof("listing all values")
	res := make([]*openreportsv1alpha1.ClusterReport, 0)
	var jsonb string

	rows, err := o.db.Query("SELECT report FROM clusterreports WHERE cluster_id = $1", o.clusterUID)
	if err != nil {
		klog.ErrorS(err, "failed to list clusterreports")
		return nil, fmt.Errorf("clusterreport list: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&jsonb); err != nil {
			klog.ErrorS(err, "failed to scan rows")
			return nil, fmt.Errorf("clusterreport list: %v", err)
		}
		var report openreportsv1alpha1.ClusterReport
		err := json.Unmarshal([]byte(jsonb), &report)
		if err != nil {
			klog.ErrorS(err, "failed to unmarshal clusterreport")
			return nil, fmt.Errorf("clusterreport list: cannot convert jsonb to clusterreport: %v", err)
		}
		res = append(res, &report)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (o *orClusterReportDB) Get(ctx context.Context, name string) (*openreportsv1alpha1.ClusterReport, error) {
	o.Lock()
	defer o.Unlock()

	var jsonb string

	row := o.db.QueryRow("SELECT report FROM clusterreports WHERE cluster_id = $1 AND name = $2", o.clusterUID, name)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("clusterreport not found name=%s", name))
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("clusterreport get %s: no such policy report", name)
		}
		return nil, fmt.Errorf("clusterreport get %s: %v", name, err)
	}
	var report openreportsv1alpha1.ClusterReport
	err := json.Unmarshal([]byte(jsonb), &report)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal report")
		return nil, fmt.Errorf("clusterreport list: cannot convert jsonb to policyreport: %v", err)
	}
	return &report, nil
}

func (o *orClusterReportDB) Create(ctx context.Context, cr *openreportsv1alpha1.ClusterReport) error {
	o.Lock()
	defer o.Unlock()

	if cr == nil {
		return errors.New("invalid cluster policy report")
	}

	klog.Infof("creating entry for key:%s", cr.Name)
	jsonb, err := json.Marshal(*cr)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal cr")
		return err
	}

	_, err = o.db.Exec("INSERT INTO clusterreports (name, report, cluster_id) VALUES ($1, $2, $3) ON CONFLICT (name, cluster_id) DO UPDATE SET report = $2", cr.Name, string(jsonb), o.clusterUID)
	if err != nil {
		klog.ErrorS(err, "failed to create cr")
		return fmt.Errorf("create clusterreport: %v", err)
	}
	return nil
}

func (o *orClusterReportDB) Update(ctx context.Context, cr *openreportsv1alpha1.ClusterReport) error {
	o.Lock()
	defer o.Unlock()

	if cr == nil {
		return errors.New("invalid cluster report")
	}

	jsonb, err := json.Marshal(*cr)
	if err != nil {
		return err
	}

	_, err = o.db.Exec("UPDATE clusterreports SET report = $1 WHERE cluster_id = $2 AND name = $3", string(jsonb), o.clusterUID, cr.Name)
	if err != nil {
		klog.ErrorS(err, "failed to updates clusterreport")
		return fmt.Errorf("update clusterreport: %v", err)
	}
	return nil
}

func (o *orClusterReportDB) Delete(ctx context.Context, name string) error {
	o.Lock()
	defer o.Unlock()

	_, err := o.db.Exec("DELETE FROM clusterreports WHERE cluster_id = $1 AND name = $2", o.clusterUID, name)
	if err != nil {
		klog.ErrorS(err, "failed to delete clusterreport")
		return fmt.Errorf("delete clusterreport: %v", err)
	}
	return nil
}
