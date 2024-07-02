package server

import (
	"fmt"

	"github.com/kyverno/reports-server/pkg/api"
	"github.com/kyverno/reports-server/pkg/storage"
	"k8s.io/component-base/metrics"
)

// RegisterMetrics registers
func RegisterMetrics(r metrics.KubeRegistry) error {
	err := RegisterServerMetrics(r.Register)
	if err != nil {
		return fmt.Errorf("unable to register server metrics: %v", err)
	}
	err = api.RegisterAPIMetrics(r.Register)
	if err != nil {
		return fmt.Errorf("unable to register API metrics: %v", err)
	}
	err = storage.RegisterStorageMetrics(r.Register)
	if err != nil {
		return fmt.Errorf("unable to register storage metrics: %v", err)
	}

	return nil
}
