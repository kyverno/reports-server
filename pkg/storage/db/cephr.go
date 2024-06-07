package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/nirmata/reports-server/pkg/storage/api"
	"k8s.io/klog/v2"
)

type cephr struct {
	sync.Mutex
	clusterId string
	db        *sql.DB
}

func NewClusterEphemeralReportStore(db *sql.DB, clusterId string) (api.ClusterEphemeralReportsInterface, error) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS clusterephemeralreports (name VARCHAR NOT NULL, clusterId VARCHAR NOT NULL, report JSONB NOT NULL, PRIMARY KEY(name, clusterId))")
	if err != nil {
		klog.ErrorS(err, "failed to create table")
		return nil, err
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS clusterephemeralreportcluster ON clusterephemeralreports(clusterId)")
	if err != nil {
		klog.ErrorS(err, "failed to create index")
		return nil, err
	}

	return &cephr{db: db, clusterId: clusterId}, nil
}

func (c *cephr) List(ctx context.Context) ([]reportsv1.ClusterEphemeralReport, error) {
	c.Lock()
	defer c.Unlock()

	klog.Infof("listing all values")
	res := make([]reportsv1.ClusterEphemeralReport, 0)
	var jsonb string

	rows, err := c.db.Query("SELECT report FROM clusterephemeralreports WHERE (clusterId = $1)", c.clusterId)
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
		res = append(res, report)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (c *cephr) Get(ctx context.Context, name string) (reportsv1.ClusterEphemeralReport, error) {
	c.Lock()
	defer c.Unlock()

	var jsonb string

	row := c.db.QueryRow("SELECT report FROM clusterephemeralreports WHERE (name = $1) AND (clusterId = $2)", name, c.clusterId)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("clusterephemeralreport not found name=%s", name))
		if err == sql.ErrNoRows {
			return reportsv1.ClusterEphemeralReport{}, fmt.Errorf("clusterephemeralreport get %s: no such ephemeral report", name)
		}
		return reportsv1.ClusterEphemeralReport{}, fmt.Errorf("clusterephemeralreport get %s: %v", name, err)
	}
	var report reportsv1.ClusterEphemeralReport
	err := json.Unmarshal([]byte(jsonb), &report)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal report")
		return reportsv1.ClusterEphemeralReport{}, fmt.Errorf("clusterephemeralreport list: cannot convert jsonb to ephemeralreport: %v", err)
	}
	return report, nil
}

func (c *cephr) Create(ctx context.Context, cephr reportsv1.ClusterEphemeralReport) error {
	c.Lock()
	defer c.Unlock()
	klog.Infof("creating entry for key:%s", cephr.Name)

	jsonb, err := json.Marshal(cephr)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal cephr")
		return err
	}

	_, err = c.db.Exec("INSERT INTO clusterephemeralreports (name, report, clusterId) VALUES ($1, $2, $3)", cephr.Name, string(jsonb), c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to crate cephr")
		return fmt.Errorf("create clusterephemeralreport: %v", err)
	}
	return nil
}

func (c *cephr) Update(ctx context.Context, cephr reportsv1.ClusterEphemeralReport) error {
	c.Lock()
	defer c.Unlock()

	jsonb, err := json.Marshal(cephr)
	if err != nil {
		return err
	}

	_, err = c.db.Exec("UPDATE clusterephemeralreports SET report = $1 WHERE (name = $2) AND (clusterId = $3)", string(jsonb), cephr.Name, c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to updates cephr")
		return fmt.Errorf("update clusterephemeralreport: %v", err)
	}
	return nil
}

func (c *cephr) Delete(ctx context.Context, name string) error {
	c.Lock()
	defer c.Unlock()

	_, err := c.db.Exec("DELETE FROM clusterephemeralreports WHERE (name = $1) AND (clusterId = $2)", name, c.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to delete cephr")
		return fmt.Errorf("delete clusterephemeralreport: %v", err)
	}
	return nil
}
