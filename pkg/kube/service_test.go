package kube

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateService(t *testing.T) {
	t.Run("service is created correctly", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		params := ServiceParams{
			Namespace:  "default",
			Name:       "test-svc",
			Port:       8080,
			TargetPort: 8080,
			Selector:   map[string]string{"app": "test-svc"},
		}
		svc, err := CreateService(ctx, client, params)
		if err != nil {
			t.Fatalf("CreateService: %v", err)
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
		if svc.Spec.Type != corev1.ServiceTypeClusterIP {
			t.Errorf("ServiceType = %v, want ClusterIP", svc.Spec.Type)
		}
		if len(svc.Spec.Ports) != 1 {
			t.Fatalf("len(Ports) = %d, want 1", len(svc.Spec.Ports))
		}
		if svc.Spec.Ports[0].Port != 8080 {
			t.Errorf("Port = %d, want 8080", svc.Spec.Ports[0].Port)
		}
		if svc.Spec.Selector["app"] != "test-svc" {
			t.Errorf("Selector = %v, want app=test-svc", svc.Spec.Selector)
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

	t.Run("service with empty namespace returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateService(ctx, client, ServiceParams{Namespace: "", Name: "x", Port: 8080})
		if err == nil {
			t.Error("CreateService with empty namespace: want error, got nil")
		}
	})

	t.Run("service with empty name returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateService(ctx, client, ServiceParams{Namespace: "default", Name: "", Port: 8080})
		if err == nil {
			t.Error("CreateService with empty name: want error, got nil")
		}
	})

	t.Run("service with zero port returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateService(ctx, client, ServiceParams{Namespace: "default", Name: "x", Port: 0})
		if err == nil {
			t.Error("CreateService with port 0: want error, got nil")
		}
	})

	t.Run("targetPort defaults to port when zero", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		params := ServiceParams{
			Namespace: "default",
			Name:      "svc",
			Port:      9090,
			Selector:  map[string]string{"app": "svc"},
		}
		svc, err := CreateService(ctx, client, params)
		if err != nil {
			t.Fatalf("CreateService: %v", err)
		}
		if svc.Spec.Ports[0].TargetPort.IntVal != 9090 {
			t.Errorf("TargetPort = %v, want 9090", svc.Spec.Ports[0].TargetPort)
		}
	})
}

func TestDeleteService(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delete-me",
			Namespace: "default",
		},
	})

	if err := DeleteService(ctx, client, "default", "delete-me"); err != nil {
		t.Fatalf("DeleteService: %v", err)
	}

	_, err := client.CoreV1().Services("default").Get(ctx, "delete-me", metav1.GetOptions{})
	if err == nil {
		t.Fatal("expected error getting deleted service, got nil")
	}
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected not found error after deleting service, got %v", err)
	}
}
