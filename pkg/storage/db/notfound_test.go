package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
)

// emptyDriver answers every query with zero rows, so QueryRow.Scan sees
// sql.ErrNoRows the same way it does when Postgres has no matching report.
type emptyDriver struct{}

func (emptyDriver) Open(string) (driver.Conn, error) { return emptyConn{}, nil }

type emptyConn struct{}

func (emptyConn) Prepare(string) (driver.Stmt, error) { return emptyStmt{}, nil }
func (emptyConn) Close() error                        { return nil }
func (emptyConn) Begin() (driver.Tx, error)           { return nil, io.EOF }

type emptyStmt struct{}

func (emptyStmt) Close() error                               { return nil }
func (emptyStmt) NumInput() int                              { return -1 }
func (emptyStmt) Exec([]driver.Value) (driver.Result, error) { return driver.RowsAffected(0), nil }
func (emptyStmt) Query([]driver.Value) (driver.Rows, error)  { return emptyRows{}, nil }

type emptyRows struct{}

func (emptyRows) Columns() []string         { return []string{"report"} }
func (emptyRows) Close() error              { return nil }
func (emptyRows) Next([]driver.Value) error { return io.EOF }

func init() {
	sql.Register("reports-server-empty", emptyDriver{})
}

func TestGetMissingReportReturnsNotFound(t *testing.T) {
	db, err := sql.Open("reports-server-empty", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	namespaced := newGenericGetter[reportsv1.EphemeralReport, *reportsv1.EphemeralReport]("ephemeralreport", utils.EphemeralReportsGR, "ephemeralreports", "cluster", db)
	_, err = namespaced.Get(ctx, "missing", "default")
	if !errors.IsNotFound(err) {
		t.Fatalf("expected NotFound from namespaced store, got %v", err)
	}
	if got := err.(errors.APIStatus).Status().Details; got.Group != utils.EphemeralReportsGR.Group || got.Kind != utils.EphemeralReportsGR.Resource || got.Name != "missing" {
		t.Fatalf("unexpected status details: %+v", got)
	}

	clustered := newGenericClusterGetter[reportsv1.ClusterEphemeralReport, *reportsv1.ClusterEphemeralReport]("clusterephemeralreport", utils.ClusterEphemeralReportsGR, "clusterephemeralreports", "cluster", db)
	_, err = clustered.Get(ctx, "missing")
	if !errors.IsNotFound(err) {
		t.Fatalf("expected NotFound from cluster store, got %v", err)
	}
	if got := err.(errors.APIStatus).Status().Details; got.Group != utils.ClusterEphemeralReportsGR.Group || got.Kind != utils.ClusterEphemeralReportsGR.Resource || got.Name != "missing" {
		t.Fatalf("unexpected status details: %+v", got)
	}
}
