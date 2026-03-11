package kube

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateDeployment(t *testing.T) {
	t.Run("deployment is created correctly", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		port := int32(8080)
		params := DeploymentParams{
			Namespace:     "default",
			Name:          "test-svc",
			Image:         "runner:latest",
			ContainerPort: port,
		}
		dep, err := CreateDeployment(ctx, client, params)
		if err != nil {
			t.Fatalf("CreateDeployment: %v", err)
		}
		if dep == nil {
			t.Fatal("CreateDeployment returned nil deployment")
		}
		if dep.Name != params.Name {
			t.Errorf("deployment.Name = %q, want %q", dep.Name, params.Name)
		}
		if dep.Namespace != params.Namespace {
			t.Errorf("deployment.Namespace = %q, want %q", dep.Namespace, params.Namespace)
		}
		if dep.Labels[LabelWorkloadTypeKey] != WorkloadTypeService {
			t.Errorf("workload-type label = %q, want %q", dep.Labels[LabelWorkloadTypeKey], WorkloadTypeService)
		}
		containers := dep.Spec.Template.Spec.Containers
		if len(containers) != 1 {
			t.Fatalf("expected 1 container, got %d", len(containers))
		}
		if containers[0].Image != params.Image {
			t.Errorf("container.Image = %q, want %q", containers[0].Image, params.Image)
		}
		if dep.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyAlways {
			t.Errorf("RestartPolicy = %q, want Always", dep.Spec.Template.Spec.RestartPolicy)
		}
		if len(containers[0].Ports) != 1 || containers[0].Ports[0].ContainerPort != port {
			t.Errorf("container port = %v, want %d", containers[0].Ports, port)
		}
		if *dep.Spec.Replicas != 1 {
			t.Errorf("Replicas = %d, want 1", *dep.Spec.Replicas)
		}
		if dep.Spec.Selector == nil || dep.Spec.Selector.MatchLabels["app"] != "test-svc" {
			t.Errorf("selector app label = %v, want app=test-svc", dep.Spec.Selector)
		}

		// check if the deployment exists in the cluster
		got, err := client.AppsV1().Deployments(params.Namespace).Get(ctx, params.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Deployments().Get: %v", err)
		}
		if got.UID != dep.UID {
			t.Error("retrieved deployment UID does not match created deployment")
		}
	})

	t.Run("deployment with empty namespace returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateDeployment(ctx, client, DeploymentParams{Namespace: "", Name: "x", Image: "img"})
		if err == nil {
			t.Error("CreateDeployment with empty namespace: want error, got nil")
		}
	})

	t.Run("deployment with empty name returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateDeployment(ctx, client, DeploymentParams{Namespace: "default", Name: "", Image: "img"})
		if err == nil {
			t.Error("CreateDeployment with empty name: want error, got nil")
		}
	})

	t.Run("deployment with empty image returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateDeployment(ctx, client, DeploymentParams{Namespace: "default", Name: "x", Image: ""})
		if err == nil {
			t.Error("CreateDeployment with empty image: want error, got nil")
		}
	})

	t.Run("deployment with ConfigMap volume has volume and volume mount", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		params := DeploymentParams{
			Namespace:      "default",
			Name:           "svc-with-cm",
			Image:          "runner:latest",
			ConfigMapName:  "my-source",
			MountPath:      "/opt/code",
			ConfigMapItems: []corev1.KeyToPath{{Key: "main.py", Path: "main.py"}},
		}
		dep, err := CreateDeployment(ctx, client, params)
		if err != nil {
			t.Fatalf("CreateDeployment: %v", err)
		}
		volumes := dep.Spec.Template.Spec.Volumes
		if len(volumes) != 1 {
			t.Fatalf("len(Volumes) = %d, want 1", len(volumes))
		}
		if volumes[0].Name != "source-code" {
			t.Errorf("volume Name = %q, want source-code", volumes[0].Name)
		}
		if volumes[0].ConfigMap == nil || volumes[0].ConfigMap.Name != "my-source" {
			t.Errorf("volume ConfigMap = %v, want Name=my-source", volumes[0].ConfigMap)
		}
		mounts := dep.Spec.Template.Spec.Containers[0].VolumeMounts
		if len(mounts) != 1 || mounts[0].Name != "source-code" || mounts[0].MountPath != "/opt/code" {
			t.Errorf("VolumeMount = %+v, want source-code /opt/code", mounts)
		}
	})
}

func TestListManagedDeployments(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "managed-dep",
				Namespace: "default",
				Labels:    map[string]string{LabelManagedKey: LabelManagedValue},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-dep",
				Namespace: "default",
			},
		},
	)
	list, err := ListManagedDeployments(ctx, client, "default")
	if err != nil {
		t.Fatalf("ListManagedDeployments: %v", err)
	}
	if len(list.Items) != 1 {
		t.Errorf("len(List.Items) = %d, want 1", len(list.Items))
	}
	if list.Items[0].Name != "managed-dep" {
		t.Errorf("Name = %q, want managed-dep", list.Items[0].Name)
	}
}

func TestDeleteDeployment(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delete-me",
			Namespace: "default",
		},
	})

	if err := DeleteDeployment(ctx, client, "default", "delete-me"); err != nil {
		t.Fatalf("DeleteDeployment: %v", err)
	}

	_, err := client.AppsV1().Deployments("default").Get(ctx, "delete-me", metav1.GetOptions{})
	if err == nil {
		t.Fatal("expected error getting deleted deployment, got nil")
	}
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected not found error after deleting deployment, got %v", err)
	}
}
