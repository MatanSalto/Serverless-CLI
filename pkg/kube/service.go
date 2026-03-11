package kube

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// ServiceParams holds parameters for creating a Service that targets a Deployment's pods.
type ServiceParams struct {
	Name       string
	Namespace  string
	Port       int32
	TargetPort int32
	Selector   map[string]string // must match the Deployment's pod template labels (e.g. app=<name>)
	// Type controls the Service type. If empty, defaults to ClusterIP.
	Type corev1.ServiceType
	// NodePort is used when Type is NodePort. If zero, Kubernetes will allocate one.
	NodePort int32
}

// CreateService creates a ClusterIP Service in the cluster targeting pods with the given selector.
func CreateService(ctx context.Context, client kubernetes.Interface, params ServiceParams) (*corev1.Service, error) {
	if params.Namespace == "" {
		return nil, errors.New("namespace is required")
	}
	if params.Name == "" {
		return nil, errors.New("name is required")
	}
	if params.Port <= 0 {
		return nil, errors.New("port is required and must be positive")
	}
	if params.Selector == nil {
		params.Selector = make(map[string]string)
	}
	targetPort := params.TargetPort
	if targetPort <= 0 {
		targetPort = params.Port
	}

	svcType := params.Type
	if svcType == "" {
		svcType = corev1.ServiceTypeClusterIP
	}

	servicePort := corev1.ServicePort{
		Name:       "http",
		Port:       params.Port,
		TargetPort: intstr.FromInt32(targetPort),
		Protocol:   corev1.ProtocolTCP,
	}
	if svcType == corev1.ServiceTypeNodePort && params.NodePort > 0 {
		servicePort.NodePort = params.NodePort
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: params.Namespace,
			Labels: map[string]string{
				LabelManagedKey:      LabelManagedValue,
				LabelWorkloadTypeKey: WorkloadTypeService,
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: params.Selector,
			Ports: []corev1.ServicePort{servicePort},
		},
	}

	return client.CoreV1().Services(params.Namespace).Create(ctx, svc, metav1.CreateOptions{})
}

// DeleteService deletes a Service by name in the given namespace.
func DeleteService(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	return client.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
