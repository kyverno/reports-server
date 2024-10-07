package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/kyverno/reports-server/pkg/storage/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type cpolrdb struct {
	sync.Mutex
	db *sql.DB
}

func NewClusterPolicyReportStore(db *sql.DB) (api.ClusterPolicyReportsInterface, error) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS clusterpolicyreports (name VARCHAR NOT NULL, report JSONB NOT NULL, PRIMARY KEY(name))")
	if err != nil {
		klog.ErrorS(err, "failed to create table")
		return nil, err
	}

	return &cpolrdb{db: db}, nil
}

func (c *cpolrdb) List(ctx context.Context) ([]*v1alpha2.ClusterPolicyReport, error) {
	c.Lock()
	defer c.Unlock()

	klog.Infof("listing all values")
	res := make([]*v1alpha2.ClusterPolicyReport, 0)
	var jsonb string

	rows, err := c.db.Query("SELECT report FROM clusterpolicyreports")
	if err != nil {
		klog.ErrorS(err, "failed to list clusterpolicyreports")
		return nil, fmt.Errorf("clusterpolicyreport list: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&jsonb); err != nil {
			klog.ErrorS(err, "failed to scan rows")
			return nil, fmt.Errorf("clusterpolicyreport list: %v", err)
		}
		var report v1alpha2.ClusterPolicyReport
		err := json.Unmarshal([]byte(jsonb), &report)
		if err != nil {
			klog.ErrorS(err, "failed to unmarshal clusterpolicyreport")
			return nil, fmt.Errorf("clusterpolicyreport list: cannot convert jsonb to clusterpolicyreport: %v", err)
		}
		res = append(res, &report)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (c *cpolrdb) Get(ctx context.Context, name string) (*v1alpha2.ClusterPolicyReport, error) {
	c.Lock()
	defer c.Unlock()

	var jsonb string

	row := c.db.QueryRow("SELECT report FROM clusterpolicyreports WHERE (name = $1)", name)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("clusterpolicyreport not found name=%s", name))
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("clusterpolicyreport get %s: no such policy report", name)
		}
		return nil, fmt.Errorf("clusterpolicyreport get %s: %v", name, err)
	}
	var report v1alpha2.ClusterPolicyReport
	err := json.Unmarshal([]byte(jsonb), &report)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal report")
		return nil, fmt.Errorf("clusterpolicyreport list: cannot convert jsonb to policyreport: %v", err)
	}
	return &report, nil
}

func (c *cpolrdb) Create(ctx context.Context, cpolr *v1alpha2.ClusterPolicyReport) error {
	c.Lock()
	defer c.Unlock()

	if cpolr == nil {
		return errors.New("invalid cluster policy report")
	}

	klog.Infof("creating entry for key:%s", cpolr.Name)
	jsonb, err := json.Marshal(*cpolr)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal cpolr")
		return err
	}

	_, err = c.db.Exec("INSERT INTO clusterpolicyreports (name, report) VALUES ($1, $2)", cpolr.Name, string(jsonb))
	if err != nil {
		klog.ErrorS(err, "failed to crate cpolr")
		return fmt.Errorf("create clusterpolicyreport: %v", err)
	}
	return nil
}

func (c *cpolrdb) Update(ctx context.Context, cpolr *v1alpha2.ClusterPolicyReport) error {
	c.Lock()
	defer c.Unlock()

	if cpolr == nil {
		return errors.New("invalid cluster policy report")
	}

	jsonb, err := json.Marshal(*cpolr)
	if err != nil {
		return err
	}

	_, err = c.db.Exec("UPDATE clusterpolicyreports SET report = $1 WHERE (name = $2)", string(jsonb), cpolr.Name)
	if err != nil {
		klog.ErrorS(err, "failed to updates cpolr")
		return fmt.Errorf("update clusterpolicyreport: %v", err)
	}
	return nil
}

func (c *cpolrdb) Delete(ctx context.Context, name string) error {
	c.Lock()
	defer c.Unlock()

	_, err := c.db.Exec("DELETE FROM clusterpolicyreports WHERE (name = $1)", name)
	if err != nil {
		klog.ErrorS(err, "failed to delete cpolr")
		return fmt.Errorf("delete clusterpolicyreport: %v", err)
	}
	return nil
}
