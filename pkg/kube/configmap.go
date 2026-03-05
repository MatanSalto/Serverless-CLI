package kube

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ConfigMapParams holds parameters for creating a ConfigMap.
type ConfigMapParams struct {
	Name      string
	Namespace string
	Data      map[string]string
}

// CreateConfigMap creates a ConfigMap in the cluster.
func CreateConfigMap(ctx context.Context, client kubernetes.Interface, params ConfigMapParams) (*corev1.ConfigMap, error) {
	if params.Namespace == "" {
		return nil, errors.New("namespace is required")
	}
	if params.Name == "" {
		return nil, errors.New("name is required")
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: params.Namespace,
		},
		Data: params.Data,
	}
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	return client.CoreV1().ConfigMaps(params.Namespace).Create(ctx, cm, metav1.CreateOptions{})
}

// ConfigMapNameForWorkload returns the source ConfigMap name for a workload (e.g. "my-job" -> "my-job-source").
func ConfigMapNameForWorkload(workloadName string) string {
	return workloadName + "-source"
}

// DeleteConfigMap deletes a ConfigMap by name in the given namespace.
func DeleteConfigMap(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	return client.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
