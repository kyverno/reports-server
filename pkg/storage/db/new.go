package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/storage/api"
	_ "github.com/lib/pq"
	openreportsv1alpha1 "github.com/openreports/reports-api/apis/openreports.io/v1alpha1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
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
	ctx := context.Background()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		klog.Infof("pinging postgres db, attempt: %d", attempt)
		err := db.PingContext(ctx)
		if err == nil {
			break
		}
		klog.Error("failed to ping db", err.Error())
		time.Sleep(sleepDuration)
	}

	err = db.PingContext(ctx)
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

	err = createOrUpdateClusterRecord(ctx, db, clusterUID, clusterName)
	if err != nil {
		klog.Error("failed to update cluster record", err.Error())
		return nil, err
	}

	err = populateClusterUIDLegacyRecords(ctx, db, clusterUID)
	if err != nil {
		klog.Error("failed to update legacy records", err.Error())
		return nil, err
	}

	klog.Info("successfully setup storage")
	return &postgresstore{
		db:                   db,
		polrstore:            newGenericGetter[v1alpha2.PolicyReport, *v1alpha2.PolicyReport]("policyreport", "policyreports", clusterUID, db),
		cpolrstore:           newGenericClusterGetter[v1alpha2.ClusterPolicyReport, *v1alpha2.ClusterPolicyReport]("clusterpolicyreport", "clusterpolicyreports", clusterUID, db),
		ephrstore:            newGenericGetter[reportsv1.EphemeralReport, *reportsv1.EphemeralReport]("ephemeralreport", "ephemeralreports", clusterUID, db),
		cephrstore:           newGenericClusterGetter[reportsv1.ClusterEphemeralReport, *reportsv1.ClusterEphemeralReport]("clusterephemeralreport", "clusterephemeralreports", clusterUID, db),
		orreportstore:        newGenericGetter[openreportsv1alpha1.Report, *openreportsv1alpha1.Report]("report", "reports", clusterUID, db),
		orclusterreportstore: newGenericClusterGetter[openreportsv1alpha1.ClusterReport, *openreportsv1alpha1.ClusterReport]("clusterreport", "clusterreports", clusterUID, db),
	}, nil
}

type postgresstore struct {
	db                   *sql.DB
	polrstore            api.GenericIface[*v1alpha2.PolicyReport]
	cpolrstore           api.GenericClusterIface[*v1alpha2.ClusterPolicyReport]
	ephrstore            api.GenericIface[*reportsv1.EphemeralReport]
	cephrstore           api.GenericClusterIface[*reportsv1.ClusterEphemeralReport]
	orreportstore        api.GenericIface[*openreportsv1alpha1.Report]
	orclusterreportstore api.GenericClusterIface[*openreportsv1alpha1.ClusterReport]
}

func (p *postgresstore) PolicyReports() api.GenericIface[*v1alpha2.PolicyReport] {
	return p.polrstore
}

func (p *postgresstore) ClusterPolicyReports() api.GenericClusterIface[*v1alpha2.ClusterPolicyReport] {
	return p.cpolrstore
}

func (p *postgresstore) EphemeralReports() api.GenericIface[*reportsv1.EphemeralReport] {
	return p.ephrstore
}

func (p *postgresstore) ClusterEphemeralReports() api.GenericClusterIface[*reportsv1.ClusterEphemeralReport] {
	return p.cephrstore
}

func (p *postgresstore) Reports() api.GenericIface[*openreportsv1alpha1.Report] {
	return p.orreportstore
}

func (p *postgresstore) ClusterReports() api.GenericClusterIface[*openreportsv1alpha1.ClusterReport] {
	return p.orclusterreportstore
}

func (p *postgresstore) Ready(ctx context.Context) bool {
	if err := p.db.PingContext(ctx); err != nil {
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

func createOrUpdateClusterRecord(ctx context.Context, db *sql.DB, clusterUID string, clusterName string) error {
	_, err := db.QueryContext(ctx, "INSERT INTO clusters (id, name) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET name = $2", clusterUID, clusterName)
	return err
}

func populateClusterUIDLegacyRecords(ctx context.Context, db *sql.DB, clusterUID string) error {
	_, err := db.QueryContext(ctx, "UPDATE clusterephemeralreports SET cluster_id = $1 WHERE cluster_id = '00000000-0000-0000-0000-000000000000'", clusterUID)
	if err != nil {
		return err
	}
	_, err = db.QueryContext(ctx, "UPDATE clusterpolicyreports SET cluster_id = $1 WHERE cluster_id = '00000000-0000-0000-0000-000000000000'", clusterUID)
	if err != nil {
		return err
	}
	_, err = db.QueryContext(ctx, "UPDATE clusterreports SET cluster_id = $1 WHERE cluster_id = '00000000-0000-0000-0000-000000000000'", clusterUID)
	if err != nil {
		return err
	}
	_, err = db.QueryContext(ctx, "UPDATE ephemeralreports SET cluster_id = $1 WHERE cluster_id = '00000000-0000-0000-0000-000000000000'", clusterUID)
	if err != nil {
		return err
	}
	_, err = db.QueryContext(ctx, "UPDATE policyreports SET cluster_id = $1 WHERE cluster_id = '00000000-0000-0000-0000-000000000000'", clusterUID)
	if err != nil {
		return err
	}
	_, err = db.QueryContext(ctx, "UPDATE reports SET cluster_id = $1 WHERE cluster_id = '00000000-0000-0000-0000-000000000000'", clusterUID)
	if err != nil {
		return err
	}
	return nil
}
