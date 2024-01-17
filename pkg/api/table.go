// Copyright 2021 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"time"

	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

func addPolicyReportToTable(table *metav1beta1.Table, polrs ...v1alpha2.PolicyReport) {
	for i, polr := range polrs {
		table.ColumnDefinitions = []metav1beta1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name", Description: "Name of the resource"},
			{Name: "Pass", Type: "integer", Format: "string"},
			{Name: "Fail", Type: "integer", Format: "string"},
			{Name: "Warn", Type: "integer", Format: "string"},
			{Name: "Error", Type: "integer", Format: "string"},
			{Name: "Skip", Type: "integer", Format: "string"},
			{Name: "Age", Type: "string", Format: "duration"},
		}
		row := make([]interface{}, 0, len(table.ColumnDefinitions))
		row = append(row, polr.Name)
		row = append(row, polr.Summary.Pass)
		row = append(row, polr.Summary.Fail)
		row = append(row, polr.Summary.Warn)
		row = append(row, polr.Summary.Error)
		row = append(row, polr.Summary.Skip)
		row = append(row, time.Since(polr.CreationTimestamp.Time).Truncate(time.Second).String())
		table.Rows = append(table.Rows, metav1beta1.TableRow{
			Cells:  row,
			Object: runtime.RawExtension{Object: &polrs[i]},
		})
	}
}

func addClusterPolicyReportToTable(table *metav1beta1.Table, cpolrs ...v1alpha2.ClusterPolicyReport) {
	for i, cpolr := range cpolrs {
		table.ColumnDefinitions = []metav1beta1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name", Description: "Name of the resource"},
			{Name: "Pass", Type: "integer", Format: "string"},
			{Name: "Fail", Type: "integer", Format: "string"},
			{Name: "Warn", Type: "integer", Format: "string"},
			{Name: "Error", Type: "integer", Format: "string"},
			{Name: "Skip", Type: "integer", Format: "string"},
			{Name: "Age", Type: "string", Format: "duration"},
		}
		row := make([]interface{}, 0, len(table.ColumnDefinitions))
		row = append(row, cpolr.Name)
		row = append(row, cpolr.Summary.Pass)
		row = append(row, cpolr.Summary.Fail)
		row = append(row, cpolr.Summary.Warn)
		row = append(row, cpolr.Summary.Error)
		row = append(row, cpolr.Summary.Skip)
		row = append(row, time.Since(cpolr.CreationTimestamp.Time).Truncate(time.Second).String())
		table.Rows = append(table.Rows, metav1beta1.TableRow{
			Cells:  row,
			Object: runtime.RawExtension{Object: &cpolrs[i]},
		})
	}
}
