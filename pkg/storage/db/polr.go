package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/kyverno/reports-server/pkg/storage/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type polrdb struct {
	sync.Mutex
	db *sql.DB
}

func NewPolicyReportStore(db *sql.DB) (api.PolicyReportsInterface, error) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS policyreports (name VARCHAR NOT NULL, namespace VARCHAR NOT NULL, report JSONB NOT NULL, PRIMARY KEY(name, namespace))")
	if err != nil {
		klog.ErrorS(err, "failed to create table")
		return nil, err
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS policyreportnamespace ON policyreports(namespace)")
	if err != nil {
		klog.ErrorS(err, "failed to create index")
		return nil, err
	}
	return &polrdb{db: db}, nil
}

func (p *polrdb) List(ctx context.Context, namespace string) ([]v1alpha2.PolicyReport, error) {
	p.Lock()
	defer p.Unlock()

	klog.Infof("listing all values for namespace:%s", namespace)
	res := make([]v1alpha2.PolicyReport, 0)
	var jsonb string

	rows, err := p.db.Query("SELECT report FROM policyreports WHERE namespace = $1", namespace)
	if err != nil {
		klog.ErrorS(err, "policyreport list: ")
		return nil, fmt.Errorf("policyreport list %q: %v", namespace, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&jsonb); err != nil {
			klog.ErrorS(err, "policyreport scan failed")
			return nil, fmt.Errorf("policyreport list %q: %v", namespace, err)
		}
		var report v1alpha2.PolicyReport
		err := json.Unmarshal([]byte(jsonb), &report)
		if err != nil {
			klog.ErrorS(err, "cannot convert jsonb to policyreport")
			return nil, fmt.Errorf("policyreport list %q: cannot convert jsonb to policyreport: %v", namespace, err)
		}
		res = append(res, report)
	}

	klog.Infof("list found length: %d", len(res))
	return res, nil
}

func (p *polrdb) Get(ctx context.Context, name, namespace string) (v1alpha2.PolicyReport, error) {
	p.Lock()
	defer p.Unlock()

	var jsonb string

	row := p.db.QueryRow("SELECT report FROM policyreports WHERE (namespace = $1) AND (name = $2)", namespace, name)
	if err := row.Scan(&jsonb); err != nil {
		if err == sql.ErrNoRows {
			klog.ErrorS(err, "policyreport not found")
			return v1alpha2.PolicyReport{}, fmt.Errorf("policyreport get %s/%s: no such policy report: %v", namespace, name, err)
		}
		klog.ErrorS(err, "policyreport not found")
		return v1alpha2.PolicyReport{}, fmt.Errorf("policyreport get %s/%s: %v", namespace, name, err)
	}
	var report v1alpha2.PolicyReport
	err := json.Unmarshal([]byte(jsonb), &report)
	if err != nil {
		klog.ErrorS(err, "cannot convert jsonb to policyreport")
		return v1alpha2.PolicyReport{}, fmt.Errorf("policyreport list %q: cannot convert jsonb to policyreport: %v", namespace, err)
	}
	return report, nil
}

func (p *polrdb) Create(ctx context.Context, polr v1alpha2.PolicyReport) error {
	p.Lock()
	defer p.Unlock()
	klog.Infof("creating entry for key:%s/%s", polr.Name, polr.Namespace)

	jsonb, err := json.Marshal(polr)
	if err != nil {
		return err
	}

	_, err = p.db.Exec("INSERT INTO policyreports (name, namespace, report) VALUES ($1, $2, $3)", polr.Name, polr.Namespace, string(jsonb))
	if err != nil {
		klog.ErrorS(err, "failed to create policy report")
		return fmt.Errorf("create policyreport: %v", err)
	}
	return nil
}

func (p *polrdb) Update(ctx context.Context, polr v1alpha2.PolicyReport) error {
	p.Lock()
	defer p.Unlock()

	jsonb, err := json.Marshal(polr)
	if err != nil {
		return err
	}

	_, err = p.db.Exec("UPDATE policyreports SET report = $1 WHERE (namespace = $2) AND (name = $3)", string(jsonb), polr.Namespace, polr.Name)
	if err != nil {
		klog.ErrorS(err, "failed to update policy report")
		return fmt.Errorf("update policyreport: %v", err)
	}
	return nil
}

func (p *polrdb) Delete(ctx context.Context, name, namespace string) error {
	p.Lock()
	defer p.Unlock()

	_, err := p.db.Exec("DELETE FROM policyreports WHERE (namespace = $1) AND (name = $2)", namespace, name)
	if err != nil {
		klog.ErrorS(err, "failed to delete policy report")
		return fmt.Errorf("delete policyreport: %v", err)
	}
	return nil
}
