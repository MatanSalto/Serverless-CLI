package kube

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateJob(t *testing.T) {

	t.Run("job is created correctly", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		params := JobParams{
			Namespace: "default",
			Name:      "test-job",
			Image:     "busybox:latest",
			Command:   []string{"echo"},
			Args:      []string{"hello"},
		}
		job, err := CreateJob(ctx, client, params)

		// Verify that the job was created correctly
		if job == nil {
			t.Errorf("CreateJob returned nil job")
		}
		if job.Name != params.Name {
			t.Errorf("job.Name = %q, want %q", job.Name, params.Name)
		}
		if job.Namespace != params.Namespace {
			t.Errorf("job.Namespace = %q, want %q", job.Namespace, params.Namespace)
		}
		if len(job.Spec.Template.Spec.Containers) != 1 {
			t.Errorf("expected 1 container, got %d", len(job.Spec.Template.Spec.Containers))
		}
		container := job.Spec.Template.Spec.Containers[0]
		if container.Image != params.Image {
			t.Errorf("container.Image = %q, want %q", container.Image, params.Image)
		}
		if job.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyNever {
			t.Errorf("RestartPolicy = %q, want Never", job.Spec.Template.Spec.RestartPolicy)
		}

		// Verify that the job is in the cluster
		got, err := client.BatchV1().Jobs(params.Namespace).Get(ctx, params.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Jobs().Get: %v", err)
		}
		if got.UID != job.UID {
			t.Errorf("retrieved job UID does not match created job")
		}
	})
	
	t.Run("job with empty namespace is not created and throws exception", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateJob(ctx, client, JobParams{Namespace: "", Name: "job-name", Image: "image"})
		if err == nil {
			t.Error("CreateJob with empty namespace: want error, got nil")
		}
	})

	t.Run("job with empty name is not created and throws exception", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateJob(ctx, client, JobParams{Namespace: "default", Name: "", Image: "image"})
		if err == nil {
			t.Error("CreateJob with empty name: want error, got nil")
		}
	})

	t.Run("job with empty image is not created and throws exception", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateJob(ctx, client, JobParams{Namespace: "default", Name: "job-name", Image: ""})
		if err == nil {
			t.Error("CreateJob with empty image: want error, got nil")
		}
	})
}
