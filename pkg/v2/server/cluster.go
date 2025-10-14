package server

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// GetClusterUID obtains the UID of the kube-system namespace
// This is used as a unique identifier for the Kubernetes cluster
func GetClusterUID(restConfig *rest.Config) (string, error) {
	if restConfig == nil {
		return "", fmt.Errorf("REST config is required to get cluster UID")
	}

	klog.V(4).Info("Getting cluster UID from kube-system namespace")

	// Create Kubernetes client
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Get kube-system namespace
	ctx := context.TODO()
	kubeSystem, err := client.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get kube-system namespace: %w", err)
	}

	clusterUID := string(kubeSystem.GetUID())
	klog.InfoS("Retrieved cluster UID", "clusterUID", clusterUID)

	return clusterUID, nil
}

// GetClusterIDOrGenerate gets the cluster UID or generates a default if not available
func GetClusterIDOrGenerate(restConfig *rest.Config, defaultID string) string {
	clusterUID, err := GetClusterUID(restConfig)
	if err != nil {
		klog.InfoS("Could not get cluster UID, using default",
			"error", err,
			"defaultID", defaultID)
		return defaultID
	}
	return clusterUID
}
