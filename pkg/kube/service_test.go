package kube

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateNodePortService(t *testing.T) {
	t.Run("nodeport service is created correctly", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		params := ServiceParams{
			Namespace:  "default",
			Name:       "nodeport-svc",
			Port:       8080,
			TargetPort: 8080,
			Selector:   map[string]string{"app": "nodeport-svc"},
			Type:       corev1.ServiceTypeNodePort,
		}
		svc, err := CreateService(ctx, client, params)
		if err != nil {
			t.Fatalf("CreateService (NodePort): %v", err)
		}
		if svc == nil {
			t.Fatal("CreateService returned nil service")
		}
		if svc.Name != params.Name {
			t.Errorf("service.Name = %q, want %q", svc.Name, params.Name)
		}
		if svc.Namespace != params.Namespace {
			t.Errorf("service.Namespace = %q, want %q", svc.Namespace, params.Namespace)
		}
		if svc.Spec.Type != corev1.ServiceTypeNodePort {
			t.Errorf("ServiceType = %v, want NodePort", svc.Spec.Type)
		}
		if len(svc.Spec.Ports) != 1 {
			t.Fatalf("len(Ports) = %d, want 1", len(svc.Spec.Ports))
		}
		if svc.Spec.Ports[0].Port != 8080 {
			t.Errorf("Port = %d, want 8080", svc.Spec.Ports[0].Port)
		}
		if svc.Spec.Selector["app"] != "nodeport-svc" {
			t.Errorf("Selector = %v, want app=nodeport-svc", svc.Spec.Selector)
		}
		if svc.Labels[LabelWorkloadTypeKey] != WorkloadTypeService {
			t.Errorf("workload-type label = %q, want %q", svc.Labels[LabelWorkloadTypeKey], WorkloadTypeService)
		}

		got, err := client.CoreV1().Services(params.Namespace).Get(ctx, params.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Services().Get: %v", err)
		}
		if got.UID != svc.UID {
			t.Error("retrieved service UID does not match created service")
		}
	})

	t.Run("nodeport service with explicit NodePort sets port in spec", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		params := ServiceParams{
			Namespace:  "default",
			Name:       "nodeport-svc-explicit",
			Port:       8080,
			TargetPort: 8080,
			Selector:   map[string]string{"app": "nodeport-svc-explicit"},
			Type:       corev1.ServiceTypeNodePort,
			NodePort:   30080,
		}
		svc, err := CreateService(ctx, client, params)
		if err != nil {
			t.Fatalf("CreateService (NodePort explicit): %v", err)
		}
		if svc.Spec.Ports[0].NodePort != 30080 {
			t.Errorf("NodePort = %d, want 30080", svc.Spec.Ports[0].NodePort)
		}
	})
}
