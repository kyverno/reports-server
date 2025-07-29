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

type ephrdb struct {
	sync.Mutex
	MultiDB   *MultiDB
	clusterId string
}

func NewEphemeralReportStore(MultiDB *MultiDB, clusterId string) (api.EphemeralReportsInterface, error) {
	_, err := MultiDB.PrimaryDB.Exec("CREATE TABLE IF NOT EXISTS ephemeralreports (name VARCHAR NOT NULL, namespace VARCHAR NOT NULL, clusterId VARCHAR NOT NULL, report JSONB NOT NULL, PRIMARY KEY(name, namespace, clusterId))")
	if err != nil {
		klog.ErrorS(err, "failed to create table")
		return nil, err
	}

	_, err = MultiDB.PrimaryDB.Exec("CREATE INDEX IF NOT EXISTS ephemeralreportnamespace ON ephemeralreports(namespace)")
	if err != nil {
		klog.ErrorS(err, "failed to create index")
		return nil, err
	}

	_, err = MultiDB.PrimaryDB.Exec("CREATE INDEX IF NOT EXISTS ephemeralreportcluster ON ephemeralreports(clusterId)")
	if err != nil {
		klog.ErrorS(err, "failed to create index")
		return nil, err
	}
	return &ephrdb{MultiDB: MultiDB, clusterId: clusterId}, nil
}

func (p *ephrdb) List(ctx context.Context, namespace string) ([]*reportsv1.EphemeralReport, error) {
	klog.Infof("listing all values for namespace:%s", namespace)
	startTime := time.Now()
	res := make([]*reportsv1.EphemeralReport, 0)
	var jsonb string
	var rows *sql.Rows
	var err error

	if len(namespace) == 0 {
		rows, err = p.MultiDB.ReadQuery(ctx, "SELECT report FROM ephemeralreports WHERE clusterId = $1", p.clusterId)
	} else {
		rows, err = p.MultiDB.ReadQuery(ctx, "SELECT report FROM ephemeralreports WHERE namespace = $1 AND clusterId = $2", namespace, p.clusterId)
	}
	if err != nil {
		klog.ErrorS(err, "ephemeralreport list: ")
		return nil, fmt.Errorf("ephemeralreport list %q: %v", namespace, err)
	}
	defer rows.Close()
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "list", "EphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "list", "EphemeralReport", time.Since(startTime))
	for rows.Next() {
		if err := rows.Scan(&jsonb); err != nil {
			klog.ErrorS(err, "ephemeralreport scan failed")
			return nil, fmt.Errorf("ephemeralreport list %q: %v", namespace, err)
		}
		var report reportsv1.EphemeralReport
		err := json.Unmarshal([]byte(jsonb), &report)
		if err != nil {
			klog.ErrorS(err, "cannot convert jsonb to ephemeralreport")
			return nil, fmt.Errorf("ephemeralreport list %q: cannot convert jsonb to ephemeralreport: %v", namespace, err)
		}
		res = append(res, &report)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (p *ephrdb) Get(ctx context.Context, name, namespace string) (*reportsv1.EphemeralReport, error) {
	startTime := time.Now()
	var jsonb string

	row := p.MultiDB.ReadQueryRow(ctx, "SELECT report FROM ephemeralreports WHERE (namespace = $1) AND (name = $2) AND (clusterId = $3)", namespace, name, p.clusterId)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("ephemeralreport not found name=%s namespace=%s", name, namespace))
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ephemeralreport get %s/%s: no such ephemeral report: %v", namespace, name, err)
		}
		return nil, fmt.Errorf("ephemeralreport get %s/%s: %v", namespace, name, err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "get", "EphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "get", "EphemeralReport", time.Since(startTime))

	var report reportsv1.EphemeralReport
	err := json.Unmarshal([]byte(jsonb), &report)
	if err != nil {
		klog.ErrorS(err, "cannot convert jsonb to ephemeralreport")
		return nil, fmt.Errorf("ephemeralreport list %q: cannot convert jsonb to ephemeralreport: %v", namespace, err)
	}
	return &report, nil
}

func (p *ephrdb) Create(ctx context.Context, polr *reportsv1.EphemeralReport) error {
	p.Lock()
	defer p.Unlock()
	startTime := time.Now()

	if polr == nil {
		return errors.New("invalid ephemeral report")
	}

	klog.Infof("creating entry for key:%s/%s", polr.Name, polr.Namespace)
	jsonb, err := json.Marshal(*polr)
	if err != nil {
		return err
	}

	_, err = p.MultiDB.PrimaryDB.Exec("INSERT INTO ephemeralreports (name, namespace, report, clusterId) VALUES ($1, $2, $3, $4) ON CONFLICT (name, namespace, clusterId) DO UPDATE SET report = EXCLUDED.report", polr.Name, polr.Namespace, string(jsonb), p.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to create ephemeral report")
		return fmt.Errorf("create ephemeralreport: %v", err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "create", "EphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "create", "EphemeralReport", time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("postgres", "create", polr, false)
	return nil
}

func (p *ephrdb) Update(ctx context.Context, polr *reportsv1.EphemeralReport) error {
	p.Lock()
	defer p.Unlock()
	startTime := time.Now()
	if polr == nil {
		return errors.New("invalid ephemeral report")
	}

	jsonb, err := json.Marshal(*polr)
	if err != nil {
		return err
	}

	_, err = p.MultiDB.PrimaryDB.Exec("UPDATE ephemeralreports SET report = $1 WHERE (namespace = $2) AND (name = $3) AND (clusterId = $4)", string(jsonb), polr.Namespace, polr.Name, p.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to update ephemeral report")
		return fmt.Errorf("update ephemeralreport: %v", err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "update", "EphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "update", "EphemeralReport", time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("postgres", "update", polr, false)
	return nil
}

func (p *ephrdb) Delete(ctx context.Context, name, namespace string) error {
	p.Lock()
	defer p.Unlock()

	report, err := p.Get(ctx, name, namespace)
	if err != nil {
		klog.ErrorS(err, "failed to get ephemeral report")
		return fmt.Errorf("delete ephemeralreport: %v", err)
	}
	startTime := time.Now()
	_, err = p.MultiDB.PrimaryDB.Exec("DELETE FROM ephemeralreports WHERE (namespace = $1) AND (name = $2) AND (clusterId = $3)", namespace, name, p.clusterId)
	if err != nil {
		klog.ErrorS(err, "failed to delete ephemeral report")
		return fmt.Errorf("delete ephemeralreport: %v", err)
	}
	serverMetrics.UpdateDBRequestTotalMetrics("postgres", "delete", "EphemeralReport")
	serverMetrics.UpdateDBRequestLatencyMetrics("postgres", "delete", "EphemeralReport", time.Since(startTime))
	storageMetrics.UpdatePolicyReportMetrics("postgres", "delete", report, false)
	return nil
}
