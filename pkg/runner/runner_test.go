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
}
