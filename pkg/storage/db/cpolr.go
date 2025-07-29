package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	serverMetrics "github.com/kyverno/reports-server/pkg/server/metrics"
	"github.com/kyverno/reports-server/pkg/storage/api"
	storageMetrics "github.com/kyverno/reports-server/pkg/storage/metrics"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type cpolrdb struct {
	sync.Mutex
	MultiDB   *MultiDB
	clusterId string
}

func NewClusterPolicyReportStore(MultiDB *MultiDB, clusterId string) (api.ClusterPolicyReportsInterface, error) {
	_, err := MultiDB.PrimaryDB.Exec("CREATE TABLE IF NOT EXISTS clusterpolicyreports (name VARCHAR NOT NULL, clusterId VARCHAR NOT NULL, report JSONB NOT NULL, PRIMARY KEY(name, clusterId))")
	if err != nil {
		klog.ErrorS(err, "failed to create table")
		return nil, err
	}

	_, err = MultiDB.PrimaryDB.Exec("CREATE INDEX IF NOT EXISTS clusterpolicyreportcluster ON clusterpolicyreports(clusterId)")
	if err != nil {
		klog.ErrorS(err, "failed to create index")
		return nil, err
	}

	return &cpolrdb{MultiDB: MultiDB, clusterId: clusterId}, nil
}

func (c *cpolrdb) List(ctx context.Context) ([]*v1alpha2.ClusterPolicyReport, error) {
	klog.Infof("listing all values")
	startTime := time.Now()
	res := make([]*v1alpha2.ClusterPolicyReport, 0)
	var jsonb string

	rows, err := c.MultiDB.ReadQuery(ctx, "SELECT report FROM clusterpolicyreports WHERE clusterId = $1", c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to list clusterpolicyreports")
		return nil, fmt.Errorf("clusterpolicyreport list: %v", err)
	}
	defer rows.Close()
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "list", "ClusterPolicyReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "list", "ClusterPolicyReport", time.Since(startTime))
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
	startTime := time.Now()
	var jsonb string

	row := c.MultiDB.ReadQueryRow(ctx, "SELECT report FROM clusterpolicyreports WHERE (name = $1) AND (clusterId = $2)", name, c.clusterId)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("clusterpolicyreport not found name=%s", name))
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("clusterpolicyreport get %s: no such policy report", name)
		}
		return nil, fmt.Errorf("clusterpolicyreport get %s: %v", name, err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "get", "ClusterPolicyReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "get", "ClusterPolicyReport", time.Since(startTime))

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
	startTime := time.Now()
	if cpolr == nil {
		return errors.New("invalid cluster policy report")
	}

	klog.Infof("creating entry for key:%s", cpolr.Name)
	jsonb, err := json.Marshal(*cpolr)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal cpolr")
		return err
	}

	_, err = c.MultiDB.PrimaryDB.Exec("INSERT INTO clusterpolicyreports (name, report, clusterId) VALUES ($1, $2, $3) ON CONFLICT (name, clusterId) DO UPDATE SET report = EXCLUDED.report", cpolr.Name, string(jsonb), c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to crate cpolr")
		return fmt.Errorf("create clusterpolicyreport: %v", err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "create", "ClusterPolicyReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "create", "ClusterPolicyReport", time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("postgres", "create", cpolr, false)
	return nil
}

func (c *cpolrdb) Update(ctx context.Context, cpolr *v1alpha2.ClusterPolicyReport) error {
	c.Lock()
	defer c.Unlock()
	startTime := time.Now()

	if cpolr == nil {
		return errors.New("invalid cluster policy report")
	}

	jsonb, err := json.Marshal(*cpolr)
	if err != nil {
		return err
	}

	_, err = c.MultiDB.PrimaryDB.Exec("UPDATE clusterpolicyreports SET report = $1 WHERE (name = $2) AND (clusterId = $3)", string(jsonb), cpolr.Name, c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to updates cpolr")
		return fmt.Errorf("update clusterpolicyreport: %v", err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "update", "ClusterPolicyReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "update", "ClusterPolicyReport", time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("postgres", "update", cpolr, false)
	return nil
}

func (c *cpolrdb) Delete(ctx context.Context, name string) error {
	c.Lock()
	defer c.Unlock()

	report, err := c.Get(ctx, name)
	if err != nil {
		klog.ErrorS(err, "failed to get cpolr")
		return fmt.Errorf("delete clusterpolicyreport: %v", err)
	}

	startTime := time.Now()
	_, err = c.MultiDB.PrimaryDB.Exec("DELETE FROM clusterpolicyreports WHERE (name = $1) AND (clusterId = $2)", name, c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to delete cpolr")
		return fmt.Errorf("delete clusterpolicyreport: %v", err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "delete", "ClusterPolicyReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "delete", "ClusterPolicyReport", time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("postgres", "delete", report, false)
	return nil
}
