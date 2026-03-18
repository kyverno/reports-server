package inmemory

import (
	"context"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	storageapi "github.com/kyverno/reports-server/pkg/storage/api"
	"github.com/kyverno/reports-server/pkg/utils"
	openreportsv1alpha1 "openreports.io/apis/openreports.io/v1alpha1"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type inMemoryDb struct {
	polrdb    *genericInMemStore[v1alpha2.PolicyReport, *v1alpha2.PolicyReport]
	cpolrdb   *genericClusterInMemStore[v1alpha2.ClusterPolicyReport, *v1alpha2.ClusterPolicyReport]
	ephrdb    *genericInMemStore[reportsv1.EphemeralReport, *reportsv1.EphemeralReport]
	cephrdb   *genericClusterInMemStore[reportsv1.ClusterEphemeralReport, *reportsv1.ClusterEphemeralReport]
	reportdb  *genericInMemStore[openreportsv1alpha1.Report, *openreportsv1alpha1.Report]
	creportdb *genericClusterInMemStore[openreportsv1alpha1.ClusterReport, *openreportsv1alpha1.ClusterReport]
}

func New() storageapi.Storage {
	return &inMemoryDb{
		polrdb:    newGenericInMemStore[v1alpha2.PolicyReport]("polr", utils.PolicyReportsGR),
		cpolrdb:   newGenericClusterInMemStore[v1alpha2.ClusterPolicyReport]("cpolr", utils.ClusterPolicyReportsGR),
		ephrdb:    newGenericInMemStore[reportsv1.EphemeralReport]("ephr", utils.EphemeralReportsGR),
		cephrdb:   newGenericClusterInMemStore[reportsv1.ClusterEphemeralReport]("cephr", utils.ClusterEphemeralReportsGR),
		reportdb:  newGenericInMemStore[openreportsv1alpha1.Report]("report", utils.OpenreportsReportGR),
		creportdb: newGenericClusterInMemStore[openreportsv1alpha1.ClusterReport]("clusterreport", utils.OpenreportsClusterReportGR),
	}
}

func (i *inMemoryDb) PolicyReports() storageapi.GenericIface[*v1alpha2.PolicyReport] {
	return i.polrdb
}

func (i *inMemoryDb) ClusterPolicyReports() storageapi.GenericClusterIface[*v1alpha2.ClusterPolicyReport] {
	return i.cpolrdb
}

func (i *inMemoryDb) EphemeralReports() storageapi.GenericIface[*reportsv1.EphemeralReport] {
	return i.ephrdb
}

func (i *inMemoryDb) ClusterEphemeralReports() storageapi.GenericClusterIface[*reportsv1.ClusterEphemeralReport] {
	return i.cephrdb
}

func (i *inMemoryDb) Reports() storageapi.GenericIface[*openreportsv1alpha1.Report] {
	return i.reportdb
}

func (i *inMemoryDb) ClusterReports() storageapi.GenericClusterIface[*openreportsv1alpha1.ClusterReport] {
	return i.creportdb
}

func (i *inMemoryDb) Ready(_ context.Context) bool {
	return true
}
