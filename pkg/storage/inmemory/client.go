package inmemory

import (
	"context"
	"sync"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	"github.com/nirmata/reports-server/pkg/storage/api"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

var groupResource = v1alpha2.SchemeGroupVersion.WithResource("policyreportsa").GroupResource()

type inMemoryDb struct {
	sync.Mutex

	cpolrdb *cpolrdb
	polrdb  *polrdb
	cephrdb *cephrdb
	ephrdb  *ephrdb
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
			db: NewDB[v1alpha2.ClusterPolicyReport](),
		},
		polrdb: &polrdb{
			db: NewDB[v1alpha2.PolicyReport](),
		},
		cephrdb: &cephrdb{
			db: NewDB[reportsv1.ClusterEphemeralReport](),
		},
		ephrdb: &ephrdb{
			db: NewDB[reportsv1.EphemeralReport](),
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

func (i *inMemoryDb) Ready() bool {
	return true
}
