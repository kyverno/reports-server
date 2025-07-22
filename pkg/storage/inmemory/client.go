package inmemory

import (
	"context"
	"sync"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/kyverno/reports-server/pkg/storage/api"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type inMemoryDb struct {
	sync.Mutex

	cpolrdb   *cpolrdb
	polrdb    *polrdb
	cephrdb   *cephrdb
	ephrdb    *ephrdb
	reportdb  *orreportdb
	creportdb *orcreportdb
}

type ClusterPolicyReportsInterface interface {
	Get(ctx context.Context, name string) (v1alpha2.ClusterPolicyReport, error)
	List(ctx context.Context, name string) ([]v1alpha2.ClusterPolicyReport, error)
	Create(ctx context.Context, cpolr v1alpha2.ClusterPolicyReport) error
	Update(ctx context.Context, cpolr v1alpha2.ClusterPolicyReport) error
	Delete(ctx context.Context, name string) error
}

func New() api.Storage {
	inMemoryDb := &inMemoryDb{
		cpolrdb: &cpolrdb{
			db: make(map[string]*v1alpha2.ClusterPolicyReport),
		},
		polrdb: &polrdb{
			db: make(map[string]*v1alpha2.PolicyReport),
		},
		cephrdb: &cephrdb{
			db: make(map[string]*reportsv1.ClusterEphemeralReport),
		},
		ephrdb: &ephrdb{
			db: make(map[string]*reportsv1.EphemeralReport),
		},
		reportdb: &orreportdb{
			db: make(map[string]*openreportsv1alpha1.Report),
		},
		creportdb: &orcreportdb{
			db: make(map[string]*openreportsv1alpha1.ClusterReport),
		},
	}
	return inMemoryDb
}

func (i *inMemoryDb) PolicyReports() api.PolicyReportsInterface {
	return i.polrdb
}

func (i *inMemoryDb) ClusterPolicyReports() api.ClusterPolicyReportsInterface {
	return i.cpolrdb
}

func (i *inMemoryDb) EphemeralReports() api.EphemeralReportsInterface {
	return i.ephrdb
}

func (i *inMemoryDb) ClusterEphemeralReports() api.ClusterEphemeralReportsInterface {
	return i.cephrdb
}

func (i *inMemoryDb) Reports() api.ReportInterface {
	return i.reportdb
}

func (i *inMemoryDb) ClusterReports() api.ClusterReportInterface {
	return i.creportdb
}

func (i *inMemoryDb) Ready() bool {
	return true
}
