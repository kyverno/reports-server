package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"k8s.io/klog/v2"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
)

type orReportDB struct {
	db         *sql.DB
	clusterUID string
}

func (o *orReportDB) List(ctx context.Context, namespace string) ([]*openreportsv1alpha1.Report, error) {
	klog.Infof("listing all values for namespace:%s", namespace)
	res := make([]*openreportsv1alpha1.Report, 0)
	var jsonb string
	var rows *sql.Rows
	var err error

	if len(namespace) == 0 {
		rows, err = o.db.Query("SELECT report FROM reports WHERE cluster_id = $1", o.clusterUID)
	} else {
		rows, err = o.db.Query("SELECT report FROM reports WHERE cluster_id = $1 AND namespace = $2", o.clusterUID, namespace)
	}

	if err != nil {
		klog.ErrorS(err, "report list: ")
		return nil, fmt.Errorf("report list %q: %v", namespace, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&jsonb); err != nil {
			klog.ErrorS(err, "report scan failed")
			return nil, fmt.Errorf("report list %q: %v", namespace, err)
		}
		var report openreportsv1alpha1.Report
		err := json.Unmarshal([]byte(jsonb), &report)
		if err != nil {
			klog.ErrorS(err, "cannot convert jsonb to report")
			return nil, fmt.Errorf("report list %q: cannot convert jsonb to report: %v", namespace, err)
		}
		res = append(res, &report)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (o *orReportDB) Get(ctx context.Context, name, namespace string) (*openreportsv1alpha1.Report, error) {
	var jsonb string

	row := o.db.QueryRow("SELECT report FROM reports WHERE cluster_id = $1 AND name = $2 AND namespace = $3", o.clusterUID, name, namespace)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("report not found name=%s namespace=%s", name, namespace))
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("report get %s/%s: no such report: %v", namespace, name, err)
		}
		return nil, fmt.Errorf("report get %s/%s: %v", namespace, name, err)
	}
	var report openreportsv1alpha1.Report
	err := json.Unmarshal([]byte(jsonb), &report)
	if err != nil {
		klog.ErrorS(err, "cannot convert jsonb to report")
		return nil, fmt.Errorf("report list %q: cannot convert jsonb to report: %v", namespace, err)
	}
	return &report, nil
}

func (o *orReportDB) Create(ctx context.Context, r *openreportsv1alpha1.Report) error {
	if r == nil {
		return errors.New("invalid report")
	}

	klog.Infof("creating entry for key:%s/%s", r.Name, r.Namespace)
	jsonb, err := json.Marshal(*r)
	if err != nil {
		return err
	}

	_, err = o.db.Exec("INSERT INTO reports (name, namespace, report, cluster_id) VALUES ($1, $2, $3, $4) ON CONFLICT (name, namespace, cluster_id) DO UPDATE SET report = $3", r.Name, r.Namespace, string(jsonb), o.clusterUID)
	if err != nil {
		klog.ErrorS(err, "failed to create report")
		return fmt.Errorf("create report: %v", err)
	}
	return nil
}

func (o *orReportDB) Update(ctx context.Context, r *openreportsv1alpha1.Report) error {
	if r == nil {
		return errors.New("invalid report")
	}

	jsonb, err := json.Marshal(*r)
	if err != nil {
		return err
	}

	_, err = o.db.Exec("UPDATE reports SET report = $1 WHERE cluster_id = $2 AND namespace = $3 AND name = $4", string(jsonb), o.clusterUID, r.Namespace, r.Name)
	if err != nil {
		klog.ErrorS(err, "failed to update report")
		return fmt.Errorf("update report: %v", err)
	}
	return nil
}

func (o *orReportDB) Delete(ctx context.Context, name, namespace string) error {
	_, err := o.db.Exec("DELETE FROM reports WHERE cluster_id = $1 AND namespace = $2 AND name = $3", o.clusterUID, namespace, name)
	if err != nil {
		klog.ErrorS(err, "failed to delete report")
		return fmt.Errorf("delete reports: %v", err)
	}
	return nil
}
