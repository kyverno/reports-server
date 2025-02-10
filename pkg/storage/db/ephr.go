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

type ephrdb struct {
	sync.Mutex
	db *sql.DB
}

func (p *ephrdb) List(ctx context.Context, namespace string) ([]*reportsv1.EphemeralReport, error) {
	p.Lock()
	defer p.Unlock()

	klog.Infof("listing all values for namespace:%s", namespace)
	res := make([]*reportsv1.EphemeralReport, 0)
	var jsonb string
	var rows *sql.Rows
	var err error

	if len(namespace) == 0 {
		rows, err = p.db.Query("SELECT report FROM ephemeralreports")
	} else {
		rows, err = p.db.Query("SELECT report FROM ephemeralreports WHERE namespace = $1", namespace)
	}
	if err != nil {
		klog.ErrorS(err, "ephemeralreport list: ")
		return nil, fmt.Errorf("ephemeralreport list %q: %v", namespace, err)
	}
	defer rows.Close()
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
	p.Lock()
	defer p.Unlock()

	var jsonb string

	row := p.db.QueryRow("SELECT report FROM ephemeralreports WHERE (namespace = $1) AND (name = $2)", namespace, name)
	if err := row.Scan(&jsonb); err != nil {
		klog.ErrorS(err, fmt.Sprintf("ephemeralreport not found name=%s namespace=%s", name, namespace))
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ephemeralreport get %s/%s: no such ephemeral report: %v", namespace, name, err)
		}
		return nil, fmt.Errorf("ephemeralreport get %s/%s: %v", namespace, name, err)
	}
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

	if polr == nil {
		return errors.New("invalid ephemeral report")
	}

	klog.Infof("creating entry for key:%s/%s", polr.Name, polr.Namespace)
	jsonb, err := json.Marshal(*polr)
	if err != nil {
		return err
	}

	_, err = p.db.Exec("INSERT INTO ephemeralreports (name, namespace, report) VALUES ($1, $2, $3)", polr.Name, polr.Namespace, string(jsonb))
	if err != nil {
		klog.ErrorS(err, "failed to create ephemeral report")
		return fmt.Errorf("create ephemeralreport: %v", err)
	}
	return nil
}

func (p *ephrdb) Update(ctx context.Context, polr *reportsv1.EphemeralReport) error {
	p.Lock()
	defer p.Unlock()

	if polr == nil {
		return errors.New("invalid ephemeral report")
	}

	jsonb, err := json.Marshal(*polr)
	if err != nil {
		return err
	}

	_, err = p.db.Exec("UPDATE ephemeralreports SET report = $1 WHERE (namespace = $2) AND (name = $3)", string(jsonb), polr.Namespace, polr.Name)
	if err != nil {
		klog.ErrorS(err, "failed to update ephemeral report")
		return fmt.Errorf("update ephemeralreport: %v", err)
	}
	return nil
}

func (p *ephrdb) Delete(ctx context.Context, name, namespace string) error {
	p.Lock()
	defer p.Unlock()

	_, err := p.db.Exec("DELETE FROM ephemeralreports WHERE (namespace = $1) AND (name = $2)", namespace, name)
	if err != nil {
		klog.ErrorS(err, "failed to delete ephemeral report")
		return fmt.Errorf("delete ephemeralreport: %v", err)
	}
	return nil
}
