package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/kyverno/reports-server/pkg/storage/inmemory"
	"github.com/kyverno/reports-server/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

// The garbage collector orphans dependents by patching them, which goes
// through Update. It only moves on when a missing object comes back as a
// NotFound status, so Update must return the status itself, not a wrapped one.
func TestUpdateMissingReportReturnsNotFoundStatus(t *testing.T) {
	store := inmemory.New()
	tests := []struct {
		name     string
		updater  rest.Updater
		resource schema.GroupResource
	}{
		{"ephemeralreport", EphemeralReportStore(store.EphemeralReports()).(rest.Updater), utils.EphemeralReportsGR},
		{"clusterephemeralreport", ClusterEphemeralReportStore(store.ClusterEphemeralReports()).(rest.Updater), utils.ClusterEphemeralReportsGR},
		{"policyreport", PolicyReportStore(store.PolicyReports()).(rest.Updater), utils.PolicyReportsGR},
		{"clusterpolicyreport", ClusterPolicyReportStore(store.ClusterPolicyReports()).(rest.Updater), utils.ClusterPolicyReportsGR},
		{"report", ReportStore(store.Reports()).(rest.Updater), utils.OpenreportsReportGR},
		{"clusterreport", ClusterReportStore(store.ClusterReports()).(rest.Updater), utils.OpenreportsClusterReportGR},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := request.WithNamespace(context.Background(), "default")
			obj, created, err := tt.updater.Update(ctx, "missing", nil, nil, nil, false, &metav1.UpdateOptions{})
			if obj != nil || created {
				t.Fatalf("expected no object, got %v (created=%v)", obj, created)
			}
			if _, ok := err.(errors.APIStatus); !ok {
				t.Fatalf("expected an API status error, got %T: %v", err, err)
			}
			if !errors.IsNotFound(err) {
				t.Fatalf("expected NotFound, got %v", err)
			}
			status := responsewriters.ErrorToAPIStatus(err)
			if status.Code != http.StatusNotFound {
				t.Fatalf("expected HTTP %d, got %d (%s)", http.StatusNotFound, status.Code, status.Message)
			}
			if status.Details == nil || status.Details.Group != tt.resource.Group || status.Details.Kind != tt.resource.Resource || status.Details.Name != "missing" {
				t.Fatalf("unexpected status details: %+v", status.Details)
			}
		})
	}
}
