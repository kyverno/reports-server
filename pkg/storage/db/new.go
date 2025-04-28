package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Import the pgx driver
	"github.com/kyverno/reports-server/pkg/storage/api"
	"k8s.io/klog/v2"
)

const (
	maxRetries    = 10
	sleepDuration = 15 * time.Second
)

func New(config *PostgresConfig, clusterId string) (api.Storage, error) {
	klog.Infof("starting postgres db")
	db, err := sql.Open("pgx", config.String())
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

	klog.Info("starting reports store")
	polrstore, err := NewPolicyReportStore(db, clusterId)
	if err != nil {
		klog.Error("failed to start policy report store", err.Error())
		return nil, err
	}

	cpolrstore, err := NewClusterPolicyReportStore(db, clusterId)
	if err != nil {
		klog.Error("failed to start cluster policy report store", err.Error())
		return nil, err
	}

	ephrstore, err := NewEphemeralReportStore(db, clusterId)
	if err != nil {
		klog.Error("failed to start policy report store", err.Error())
		return nil, err
	}

	cephrstore, err := NewClusterEphemeralReportStore(db, clusterId)
	if err != nil {
		klog.Error("failed to start cluster policy report store", err.Error())
		return nil, err
	}

	klog.Info("successfully setup storage")
	return &postgresstore{
		db:         db,
		polrstore:  polrstore,
		cpolrstore: cpolrstore,
		ephrstore:  ephrstore,
		cephrstore: cephrstore,
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
	if p.Port != 0 {
		hosts := strings.Split(p.Host, ",")
		for i, host := range hosts {
			hosts[i] = fmt.Sprintf("%s:%d", host, p.Port)
		}
		p.Host = strings.Join(hosts, ",")
	}

	// This is to handle special characters in the user and password
	encodedUser := url.QueryEscape(p.User)
	encodedPassword := url.QueryEscape(p.Password)

	urlStr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		encodedUser, encodedPassword, p.Host, p.DBname, p.SSLMode)

	if p.SSLRootCert != "" {
		urlStr += fmt.Sprintf("&sslrootcert=%s", p.SSLRootCert)
	}
	if p.SSLKey != "" {
		urlStr += fmt.Sprintf("&sslkey=%s", p.SSLKey)
	}
	if p.SSLCert != "" {
		urlStr += fmt.Sprintf("&sslcert=%s", p.SSLCert)
	}

	return urlStr
}
