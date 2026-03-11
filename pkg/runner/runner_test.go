package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"serverless-cli/pkg/kube"
)

func TestRunSource(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.py")
	if err := os.WriteFile(mainPath, []byte("print(1)"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := RunSource(ctx, client, RunSourceParams{
		SourcePath:  dir,
		Namespace:   "default",
		JobName:     "test-job",
		RunnerImage: "runner:test",
		Entrypoint:  "main.py",
	})
	if err != nil {
		t.Fatalf("RunSource: %v", err)
	}

	cm, err := client.CoreV1().ConfigMaps("default").Get(ctx, "test-job-source", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get ConfigMap: %v", err)
	}
	if len(cm.Data) != 1 {
		t.Errorf("ConfigMap has %d keys, want 1", len(cm.Data))
	}
	if cm.Data["main.py"] != "print(1)" {
		t.Errorf("ConfigMap main.py = %q, want %q", cm.Data["main.py"], "print(1)")
	}

	job, err := client.BatchV1().Jobs("default").Get(ctx, "test-job", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get Job: %v", err)
	}
	if len(job.Spec.Template.Spec.Volumes) != 1 || job.Spec.Template.Spec.Volumes[0].ConfigMap.Name != "test-job-source" {
		t.Errorf("Job volume not configured from ConfigMap test-job-source")
	}
	env := job.Spec.Template.Spec.Containers[0].Env
	var found bool
	for _, e := range env {
		if e.Name == "SLP_ENTRYPOINT" && e.Value == "main.py" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("SLP_ENTRYPOINT env not set to main.py: %v", env)
	}
}

func TestRunSource_exceedsSizeLimit(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	dir := t.TempDir()
	// Create a file larger than 1 MiB
	big := make([]byte, ConfigMapMaxSize+1)
	fpath := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(fpath, big, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := RunSource(ctx, client, RunSourceParams{
		SourcePath:  dir,
		Namespace:   "default",
		JobName:     "big-job",
		RunnerImage: "runner:test",
		Entrypoint:  "big.txt",
	})
	if err == nil {
		t.Fatal("RunSource: expected error for oversized source, got nil")
	}
}

func TestCleanupSource(t *testing.T) {
	t.Run("cleanup source", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset(
			&batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cleanup-job",
					Namespace: "default",
				},
			},
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cleanup-job-source",
					Namespace: "default",
				},
				Data: map[string]string{"main.py": "print(1)"},
			},
		)

		if err := CleanupSource(ctx, client, "default", "cleanup-job"); err != nil {
			t.Fatalf("CleanupSource: %v", err)
		}

		// Job should be gone
		if _, err := client.BatchV1().Jobs("default").Get(ctx, "cleanup-job", metav1.GetOptions{}); err == nil {
			t.Fatal("expected error getting deleted job, got nil")
		} else if !apierrors.IsNotFound(err) {
			t.Fatalf("expected not found error for job, got %v", err)
		}

		// ConfigMap should be gone
		if _, err := client.CoreV1().ConfigMaps("default").Get(ctx, "cleanup-job-source", metav1.GetOptions{}); err == nil {
			t.Fatal("expected error getting deleted configmap, got nil")
		} else if !apierrors.IsNotFound(err) {
			t.Fatalf("expected not found error for configmap, got %v", err)
		}
	})
	t.Run("cleanup source with missing configmap is ignored", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset(
			&batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cleanup-job-no-cm",
					Namespace: "default",
				},
			},
		)

		if err := CleanupSource(ctx, client, "default", "cleanup-job-no-cm"); err != nil {
			t.Fatalf("CleanupSource (no configmap): %v", err)
		}
	})
	t.Run("cleanup source with deployment and service", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cleanup-svc-source",
					Namespace: "default",
				},
				Data: map[string]string{"main.py": "print(1)"},
			},
		)
		// Create deployment and service via kube package so we have valid objects
		_, _ = kube.CreateDeployment(ctx, client, kube.DeploymentParams{
			Name: "cleanup-svc", Namespace: "default", Image: "runner:test", ContainerPort: 8080,
		})
		_, _ = kube.CreateService(ctx, client, kube.ServiceParams{
			Name: "cleanup-svc", Namespace: "default", Port: 8080, Selector: map[string]string{"app": "cleanup-svc"},
		})

		if err := CleanupSource(ctx, client, "default", "cleanup-svc"); err != nil {
			t.Fatalf("CleanupSource: %v", err)
		}

		if _, err := client.AppsV1().Deployments("default").Get(ctx, "cleanup-svc", metav1.GetOptions{}); err == nil {
			t.Fatal("expected deployment to be deleted")
		} else if !apierrors.IsNotFound(err) {
			t.Fatalf("expected not found for deployment, got %v", err)
		}
		if _, err := client.CoreV1().Services("default").Get(ctx, "cleanup-svc", metav1.GetOptions{}); err == nil {
			t.Fatal("expected service to be deleted")
		} else if !apierrors.IsNotFound(err) {
			t.Fatalf("expected not found for service, got %v", err)
		}
		if _, err := client.CoreV1().ConfigMaps("default").Get(ctx, "cleanup-svc-source", metav1.GetOptions{}); err == nil {
			t.Fatal("expected configmap to be deleted")
		} else if !apierrors.IsNotFound(err) {
			t.Fatalf("expected not found for configmap, got %v", err)
		}
	})
}

func TestRunServiceSource(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.py")
	if err := os.WriteFile(mainPath, []byte("print(1)"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	svc, err := RunServiceSource(ctx, client, RunServiceSourceParams{
		SourcePath:  dir,
		Namespace:   "default",
		ServiceName: "test-svc",
		RunnerImage: "runner:test",
		Entrypoint:  "main.py",
		Port:        8080,
	})
	if err != nil {
		t.Fatalf("RunServiceSource: %v", err)
	}

	if svc == nil {
		t.Fatal("RunServiceSource returned nil service")
	}

	cm, err := client.CoreV1().ConfigMaps("default").Get(ctx, "test-svc-source", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get ConfigMap: %v", err)
	}
	if cm.Data["main.py"] != "print(1)" {
		t.Errorf("ConfigMap main.py = %q, want %q", cm.Data["main.py"], "print(1)")
	}

	dep, err := client.AppsV1().Deployments("default").Get(ctx, "test-svc", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get Deployment: %v", err)
	}
	if dep.Labels[kube.LabelWorkloadTypeKey] != kube.WorkloadTypeService {
		t.Errorf("Deployment workload-type = %q, want %q", dep.Labels[kube.LabelWorkloadTypeKey], kube.WorkloadTypeService)
	}
	if len(dep.Spec.Template.Spec.Containers[0].Ports) != 1 || dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != 8080 {
		t.Errorf("Deployment container port not 8080: %v", dep.Spec.Template.Spec.Containers[0].Ports)
	}
	env := dep.Spec.Template.Spec.Containers[0].Env
	var hasEntrypoint, hasPort bool
	for _, e := range env {
		if e.Name == "SLP_ENTRYPOINT" && e.Value == "main.py" {
			hasEntrypoint = true
		}
		if e.Name == "SLP_PORT" && e.Value == "8080" {
			hasPort = true
		}
	}
	if !hasEntrypoint {
		t.Errorf("SLP_ENTRYPOINT not set: %v", env)
	}
	if !hasPort {
		t.Errorf("SLP_PORT not set to 8080: %v", env)
	}

	svcFromClient, err := client.CoreV1().Services("default").Get(ctx, "test-svc", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get Service: %v", err)
	}
	if svcFromClient.Spec.Ports[0].Port != 8080 {
		t.Errorf("Service port = %d, want 8080", svcFromClient.Spec.Ports[0].Port)
	}
	if svcFromClient.Spec.Selector["app"] != "test-svc" {
		t.Errorf("Service selector = %v, want app=test-svc", svcFromClient.Spec.Selector)
	}
}

func TestRunServiceSource_exceedsSizeLimit(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	dir := t.TempDir()
	big := make([]byte, ConfigMapMaxSize+1)
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), big, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := RunServiceSource(ctx, client, RunServiceSourceParams{
		SourcePath:  dir,
		Namespace:   "default",
		ServiceName: "big-svc",
		RunnerImage: "runner:test",
		Entrypoint:  "big.txt",
		Port:        8080,
	})
	if err == nil {
		t.Fatal("RunServiceSource: expected error for oversized source, got nil")
	}
}
