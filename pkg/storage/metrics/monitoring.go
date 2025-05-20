// Copyright 2020 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"strconv"

	reportsv1 "github.com/kyverno/kyverno/api/reports/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/component-base/metrics"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

var policyReportsTotal = metrics.NewCounterVec(
	&metrics.CounterOpts{
		Namespace: "reports_server",
		Subsystem: "storage",
		Name:      "policy_reports_total",
		Help:      "Total number of policy reports",
	},
	[]string{"type", "namespace", "resourceName", "resourceKind", "reportType", "operation", "migratedResource"},
)

func RegisterStorageMetrics(registrationFunc func(metrics.Registerable) error) error {
	return registrationFunc(policyReportsTotal)
}

// UpdatePolicyReportMetrics updates the policy reports metrics based on the operation
func UpdatePolicyReportMetrics(dbType string, operation string, report any, isMigrated bool) {
	switch r := report.(type) {
	case *v1alpha2.PolicyReport:
		resourceName, resourceKind := getResourceNameAndKindFromOwnerReferences(r.OwnerReferences)
		updatePolicyReportMetric(dbType, operation, r.Namespace, resourceName, resourceKind, "PolicyReport", strconv.FormatBool(isMigrated))
	case *v1alpha2.ClusterPolicyReport:
		resourceName, resourceKind := getResourceNameAndKindFromOwnerReferences(r.OwnerReferences)
		updatePolicyReportMetric(dbType, operation, "", resourceName, resourceKind, "ClusterPolicyReport", strconv.FormatBool(isMigrated))
	case *reportsv1.EphemeralReport:
		resourceName, resourceKind := getResourceNameAndKindFromOwnerReferences(r.OwnerReferences)
		updatePolicyReportMetric(dbType, operation, r.Namespace, resourceName, resourceKind, "EphemeralReport", strconv.FormatBool(isMigrated))
	case *reportsv1.ClusterEphemeralReport:
		resourceName, resourceKind := getResourceNameAndKindFromOwnerReferences(r.OwnerReferences)
		updatePolicyReportMetric(dbType, operation, "", resourceName, resourceKind, "ClusterEphemeralReport", strconv.FormatBool(isMigrated))
	}
}

func updatePolicyReportMetric(dbType, operation, namespace, resourceName, resourceKind, reportType, migratedResource string) {

	policyReportsTotal.WithLabelValues(
		dbType,
		namespace,
		resourceName,
		resourceKind,
		reportType,
		operation,
		migratedResource,
	).Inc()
}

func getResourceNameAndKindFromOwnerReferences(ownerReferences []v1.OwnerReference) (string, string) {
	if len(ownerReferences) == 0 {
		return "", ""
	}
	return ownerReferences[0].Name, ownerReferences[0].Kind
}
