package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kyverno/reports-server/pkg/storage/api"
	_ "github.com/lib/pq"
	"k8s.io/klog/v2"
)

const (
	maxRetries    = 10
	sleepDuration = 15 * time.Second
)

func New(config *PostgresConfig) (api.Storage, error) {
	klog.Infof("starting postgres db")
	db, err := sql.Open("postgres", config.String())
	if err != nil {
		klog.Error("failed to open db", err.Error())
		return nil, err
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		klog.Infof("pinging postgres db, attempt: %d", attempt)
		err := db.PingContext(context.TODO())
		if err == nil {
			break
		}
		klog.Error("failed to ping db", err.Error())
		time.Sleep(sleepDuration)
	}

	err = db.Ping()
	if err != nil {
		klog.Error("failed to ping db", err.Error())
		return nil, err
	}

	klog.Info("successfully connected to db")

	err = RunDatabaseMigration(db, config.DBname)
	if err != nil {
		klog.Error("failed to perform db migration", err.Error())
		return nil, err
	}

	klog.Info("successfully setup storage")
	return &postgresstore{
		db:         db,
		polrstore:  &polrdb{db: db},
		cpolrstore: &cpolrdb{db: db},
		ephrstore:  &ephrdb{db: db},
		cephrstore: &cephr{db: db},
	}, nil
}

type postgresstore struct {
	db         *sql.DB
	polrstore  api.PolicyReportsInterface
	cpolrstore api.ClusterPolicyReportsInterface
	ephrstore  api.EphemeralReportsInterface
	cephrstore api.ClusterEphemeralReportsInterface
}

func (p *postgresstore) ClusterPolicyReports() api.ClusterPolicyReportsInterface {
	return p.cpolrstore
}

func (p *postgresstore) PolicyReports() api.PolicyReportsInterface {
	return p.polrstore
}

func (p *postgresstore) ClusterEphemeralReports() api.ClusterEphemeralReportsInterface {
	return p.cephrstore
}

func (p *postgresstore) EphemeralReports() api.EphemeralReportsInterface {
	return p.ephrstore
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
