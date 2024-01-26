package db

import (
	"database/sql"
	"fmt"

	"github.com/kyverno/reports-server/pkg/storage/api"
	_ "github.com/lib/pq"
	"k8s.io/klog/v2"
)

func New(config *PostgresConfig) (api.Storage, error) {
	klog.Infof("starting postgres db, config: %s", config.String())
	db, err := sql.Open("postgres", config.String())
	if err != nil {
		klog.Error("failed to open db", err.Error())
		return nil, err
	}

	klog.Info("pinging postgres db")
	err = db.Ping()
	if err != nil {
		klog.Error("failed to ping db", err.Error())
		return nil, err
	}

	klog.Info("successfully connected to db")

	klog.Info("starting reports store")
	polrstore, err := NewPolicyReportStore(db)
	if err != nil {
		klog.Error("failed to start policy report store", err.Error())
		return nil, err
	}

	cpolrstore, err := NewClusterPolicyReportStore(db)
	if err != nil {
		klog.Error("failed to start cluster policy report store", err.Error())
		return nil, err
	}

	klog.Info("successfully setup storage")
	return &postgresstore{
		db:         db,
		polrstore:  polrstore,
		cpolrstore: cpolrstore,
	}, nil
}

type postgresstore struct {
	db         *sql.DB
	polrstore  api.PolicyReportsInterface
	cpolrstore api.ClusterPolicyReportsInterface
}

func (p *postgresstore) ClusterPolicyReports() api.ClusterPolicyReportsInterface {
	return p.cpolrstore
}

func (p *postgresstore) PolicyReports() api.PolicyReportsInterface {
	return p.polrstore
}

func (p *postgresstore) Ready() bool {
	if err := p.db.Ping(); err != nil {
		klog.Error("failed to ping db", err.Error())
		return false
	}
	return true
}

type PostgresConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DBname      string
	SSLMode     string
	SSLRootCert string
	SSLKey      string
	SSLCert     string
}

func (p PostgresConfig) String() string {
	return fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=%s sslrootcert=%s sslkey=%s sslcert=%s",
		p.Host, p.Port, p.User, p.Password, p.DBname, p.SSLMode, p.SSLRootCert, p.SSLKey, p.SSLCert)
}
