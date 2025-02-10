package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"k8s.io/klog/v2"
)

type cephr struct {
	sync.Mutex
	db         *sql.DB
	clusterUID string
}

func (c *cephr) List(ctx context.Context) ([]*reportsv1.ClusterEphemeralReport, error) {
	c.Lock()
	defer c.Unlock()

	klog.Infof("listing all values")
	res := make([]*reportsv1.ClusterEphemeralReport, 0)
	var jsonb string

	rows, err := c.db.Query("SELECT report FROM clusterephemeralreports WHERE cluster_id = $1", c.clusterUID)
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
	return res, nil
}

func (c *cephr) Get(ctx context.Context, name string) (*reportsv1.ClusterEphemeralReport, error) {
	c.Lock()
	defer c.Unlock()

	var jsonb string

	row := c.db.QueryRow("SELECT report FROM clusterephemeralreports WHERE cluster_id = $1 AND name = $2", c.clusterUID, name)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("clusterephemeralreport not found name=%s", name))
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("clusterephemeralreport get %s: no such ephemeral report", name)
		}
		return nil, fmt.Errorf("clusterephemeralreport get %s: %v", name, err)
	}
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

	if cephr == nil {
		return errors.New("invalid cluster ephemeral report")
	}

	klog.Infof("creating entry for key:%s", cephr.Name)
	jsonb, err := json.Marshal(*cephr)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal cephr")
		return err
	}

	_, err = c.db.Exec("INSERT INTO clusterephemeralreports (name, report, cluster_id) VALUES ($1, $2, $3) ON CONFLICT (name, cluster_id) DO UPDATE SET report = $2", cephr.Name, string(jsonb), c.clusterUID)
	if err != nil {
		klog.ErrorS(err, "failed to crate cephr")
		return fmt.Errorf("create clusterephemeralreport: %v", err)
	}
	return nil
}

func (c *cephr) Update(ctx context.Context, cephr *reportsv1.ClusterEphemeralReport) error {
	c.Lock()
	defer c.Unlock()

	if cephr == nil {
		return errors.New("invalid cluster ephemeral report")
	}

	jsonb, err := json.Marshal(*cephr)
	if err != nil {
		return err
	}

	_, err = c.db.Exec("UPDATE clusterephemeralreports SET report = $1 WHERE cluster_id = $2 AND name = $3", string(jsonb), c.clusterUID, cephr.Name)
	if err != nil {
		klog.ErrorS(err, "failed to updates cephr")
		return fmt.Errorf("update clusterephemeralreport: %v", err)
	}
	return nil
}

func (c *cephr) Delete(ctx context.Context, name string) error {
	c.Lock()
	defer c.Unlock()

	_, err := c.db.Exec("DELETE FROM clusterephemeralreports WHERE cluster_id = $1 AND name = $2", c.clusterUID, name)
	if err != nil {
		klog.ErrorS(err, "failed to delete cephr")
		return fmt.Errorf("delete clusterephemeralreport: %v", err)
	}
	return nil
}
