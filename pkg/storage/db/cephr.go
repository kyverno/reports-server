package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	serverMetrics "github.com/kyverno/reports-server/pkg/server/metrics"
	"github.com/kyverno/reports-server/pkg/storage/api"
	storageMetrics "github.com/kyverno/reports-server/pkg/storage/metrics"
	"k8s.io/klog/v2"
)

type cephr struct {
	sync.Mutex
	clusterId string
	MultiDB   *MultiDB
}

func NewClusterEphemeralReportStore(MultiDB *MultiDB, clusterId string) (api.ClusterEphemeralReportsInterface, error) {
	_, err := MultiDB.PrimaryDB.Exec("CREATE TABLE IF NOT EXISTS clusterephemeralreports (name VARCHAR NOT NULL, clusterId VARCHAR NOT NULL, report JSONB NOT NULL, PRIMARY KEY(name, clusterId))")
	if err != nil {
		klog.ErrorS(err, "failed to create table")
		return nil, err
	}

	_, err = MultiDB.PrimaryDB.Exec("CREATE INDEX IF NOT EXISTS clusterephemeralreportcluster ON clusterephemeralreports(clusterId)")
	if err != nil {
		klog.ErrorS(err, "failed to create index")
		return nil, err
	}

	return &cephr{MultiDB: MultiDB, clusterId: clusterId}, nil
}

func (c *cephr) List(ctx context.Context) ([]*reportsv1.ClusterEphemeralReport, error) {
	klog.Infof("listing all values")
	startTime := time.Now()
	res := make([]*reportsv1.ClusterEphemeralReport, 0)
	var jsonb string

	rows, err := c.MultiDB.ReadQuery(ctx, "SELECT report FROM clusterephemeralreports WHERE (clusterId = $1)", c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to list clusterephemeralreports")
		return nil, fmt.Errorf("clusterephemeralreports list: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&jsonb); err != nil {
			klog.ErrorS(err, "failed to scan rows")
			return nil, fmt.Errorf("clusterephemeralreports list: %v", err)
		}
		var report reportsv1.ClusterEphemeralReport
		err := json.Unmarshal([]byte(jsonb), &report)
		if err != nil {
			klog.ErrorS(err, "failed to unmarshal clusterephemeralreports")
			return nil, fmt.Errorf("clusterephemeralreports list: cannot convert jsonb to clusterephemeralreports: %v", err)
		}
		res = append(res, &report)
	}

	klog.Infof("list found length: %d", len(res))
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "list", "ClusterEphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "list", "ClusterEphemeralReport", time.Since(startTime))
	return res, nil
}

func (c *cephr) Get(ctx context.Context, name string) (*reportsv1.ClusterEphemeralReport, error) {
	startTime := time.Now()
	var jsonb string

	row := c.MultiDB.ReadQueryRow(ctx, "SELECT report FROM clusterephemeralreports WHERE (name = $1) AND (clusterId = $2)", name, c.clusterId)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("clusterephemeralreport not found name=%s", name))
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("clusterephemeralreport get %s: no such ephemeral report", name)
		}
		return nil, fmt.Errorf("clusterephemeralreport get %s: %v", name, err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "get", "ClusterEphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "get", "ClusterEphemeralReport", time.Since(startTime))

	var report reportsv1.ClusterEphemeralReport
	err := json.Unmarshal([]byte(jsonb), &report)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal report")
		return nil, fmt.Errorf("clusterephemeralreport list: cannot convert jsonb to ephemeralreport: %v", err)
	}
	return &report, nil
}

func (c *cephr) Create(ctx context.Context, cephr *reportsv1.ClusterEphemeralReport) error {
	c.Lock()
	defer c.Unlock()
	startTime := time.Now()

	if cephr == nil {
		return errors.New("invalid cluster ephemeral report")
	}

	klog.Infof("creating entry for key:%s", cephr.Name)
	jsonb, err := json.Marshal(*cephr)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal cephr")
		return err
	}

	_, err = c.MultiDB.PrimaryDB.Exec("INSERT INTO clusterephemeralreports (name, report, clusterId) VALUES ($1, $2, $3)", cephr.Name, string(jsonb), c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to crate cephr")
		return fmt.Errorf("create clusterephemeralreport: %v", err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "create", "ClusterEphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "create", "ClusterEphemeralReport", time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("postgres", "create", cephr, false)
	return nil
}

func (c *cephr) Update(ctx context.Context, cephr *reportsv1.ClusterEphemeralReport) error {
	c.Lock()
	defer c.Unlock()
	startTime := time.Now()

	if cephr == nil {
		return errors.New("invalid cluster ephemeral report")
	}

	jsonb, err := json.Marshal(*cephr)
	if err != nil {
		return err
	}

	_, err = c.MultiDB.PrimaryDB.Exec("UPDATE clusterephemeralreports SET report = $1 WHERE (name = $2) AND (clusterId = $3)", string(jsonb), cephr.Name, c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to updates cephr")
		return fmt.Errorf("update clusterephemeralreport: %v", err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "update", "ClusterEphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "update", "ClusterEphemeralReport", time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("postgres", "update", cephr, false)
	return nil
}

func (c *cephr) Delete(ctx context.Context, name string) error {
	c.Lock()
	defer c.Unlock()
	startTime := time.Now()

	report, err := c.Get(ctx, name)
	if err != nil {
		klog.ErrorS(err, "failed to get cephr")
		return fmt.Errorf("delete clusterephemeralreport: %v", err)
	}

	_, err = c.MultiDB.PrimaryDB.Exec("DELETE FROM clusterephemeralreports WHERE (name = $1) AND (clusterId = $2)", name, c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to delete cephr")
		return fmt.Errorf("delete clusterephemeralreport: %v", err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "delete", "ClusterEphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "delete", "ClusterEphemeralReport", time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("postgres", "delete", report, false)
	return nil
}
