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

func New(config *PostgresConfig, clusterUID string, clusterName string) (api.Storage, error) {
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

	err = createOrUpdateClusterRecord(db, clusterUID, clusterName)
	if err != nil {
		klog.Error("failed to update cluster record", err.Error())
		return nil, err
	}

	err = populateClusterUIDLegacyRecords(db, clusterUID)
	if err != nil {
		klog.Error("failed to update legacy records", err.Error())
		return nil, err
	}

	klog.Info("successfully setup storage")
	return &postgresstore{
		db:         db,
		polrstore:  &polrdb{db: db, clusterUID: clusterUID},
		cpolrstore: &cpolrdb{db: db, clusterUID: clusterUID},
		ephrstore:  &ephrdb{db: db, clusterUID: clusterUID},
		cephrstore: &cephr{db: db, clusterUID: clusterUID},
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

func createOrUpdateClusterRecord(db *sql.DB, clusterUID string, clusterName string) error {
	_, err := db.Query("INSERT INTO clusters (id, name) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET name = $2", clusterUID, clusterName)
	return err
}

func populateClusterUIDLegacyRecords(db *sql.DB, clusterUID string) error {
	_, err := db.Query("UPDATE clusterephemeralreports SET cluster_id = $1 WHERE cluster_id = '00000000-0000-0000-0000-000000000000'", clusterUID)
	if err != nil {
		return err
	}
	_, err = db.Query("UPDATE clusterpolicyreports SET cluster_id = $1 WHERE cluster_id = '00000000-0000-0000-0000-000000000000'", clusterUID)
	if err != nil {
		return err
	}
	_, err = db.Query("UPDATE ephemeralreports SET cluster_id = $1 WHERE cluster_id = '00000000-0000-0000-0000-000000000000'", clusterUID)
	if err != nil {
		return err
	}
	_, err = db.Query("UPDATE policyreports SET cluster_id = $1 WHERE cluster_id = '00000000-0000-0000-0000-000000000000'", clusterUID)
	if err != nil {
		return err
	}
	return nil
}
