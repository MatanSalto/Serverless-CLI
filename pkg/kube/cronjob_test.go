package kube

import (
	"context"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateCronJob(t *testing.T) {
	t.Run("cronjob is created correctly", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		params := CronJobParams{
			Namespace: "default",
			Name:      "test-cron",
			Schedule:  "0 * * * *",
			Image:     "busybox:latest",
			Command:   []string{"echo"},
			Args:      []string{"hello"},
		}
		cronJob, err := CreateCronJob(ctx, client, params)
		if err != nil {
			t.Fatalf("CreateCronJob: %v", err)
		}
		if cronJob == nil {
			t.Fatal("CreateCronJob returned nil cronjob")
		}
		if cronJob.Name != params.Name {
			t.Errorf("cronJob.Name = %q, want %q", cronJob.Name, params.Name)
		}
		if cronJob.Namespace != params.Namespace {
			t.Errorf("cronJob.Namespace = %q, want %q", cronJob.Namespace, params.Namespace)
		}
		if cronJob.Spec.Schedule != params.Schedule {
			t.Errorf("cronJob.Spec.Schedule = %q, want %q", cronJob.Spec.Schedule, params.Schedule)
		}
		if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers) != 1 {
			t.Errorf("expected 1 container, got %d", len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers))
		}
		container := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
		if container.Image != params.Image {
			t.Errorf("container.Image = %q, want %q", container.Image, params.Image)
		}
		if cronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyNever {
			t.Errorf("RestartPolicy = %q, want Never", cronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy)
		}
		if cronJob.Labels[LabelWorkloadTypeKey] != WorkloadTypeCron {
			t.Errorf("workload-type label = %q, want %q", cronJob.Labels[LabelWorkloadTypeKey], WorkloadTypeCron)
		}

		got, err := client.BatchV1().CronJobs(params.Namespace).Get(ctx, params.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("CronJobs().Get: %v", err)
		}
		if got.UID != cronJob.UID {
			t.Error("retrieved cronjob UID does not match created cronjob")
		}
	})

	t.Run("cronjob with empty namespace returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateCronJob(ctx, client, CronJobParams{Namespace: "", Name: "cron", Schedule: "0 0 * * *", Image: "img"})
		if err == nil {
			t.Error("CreateCronJob with empty namespace: want error, got nil")
		}
	})

	t.Run("cronjob with empty name returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateCronJob(ctx, client, CronJobParams{Namespace: "default", Name: "", Schedule: "0 0 * * *", Image: "img"})
		if err == nil {
			t.Error("CreateCronJob with empty name: want error, got nil")
		}
	})

	t.Run("cronjob with empty schedule returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateCronJob(ctx, client, CronJobParams{Namespace: "default", Name: "cron", Schedule: "", Image: "img"})
		if err == nil {
			t.Error("CreateCronJob with empty schedule: want error, got nil")
		}
	})

	t.Run("cronjob with empty image returns error", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		_, err := CreateCronJob(ctx, client, CronJobParams{Namespace: "default", Name: "cron", Schedule: "0 0 * * *", Image: ""})
		if err == nil {
			t.Error("CreateCronJob with empty image: want error, got nil")
		}
	})

	t.Run("cronjob with ConfigMap volume has volume and volume mount", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		params := CronJobParams{
			Namespace:      "default",
			Name:           "cron-with-cm",
			Schedule:       "0 * * * *",
			Image:          "runner:latest",
			ConfigMapName:  "my-source",
			MountPath:      "/opt/code",
			ConfigMapItems: []corev1.KeyToPath{{Key: "main.py", Path: "main.py"}},
		}
		cronJob, err := CreateCronJob(ctx, client, params)
		if err != nil {
			t.Fatalf("CreateCronJob: %v", err)
		}
		podSpec := cronJob.Spec.JobTemplate.Spec.Template.Spec
		if len(podSpec.Volumes) != 1 {
			t.Fatalf("len(Volumes) = %d, want 1", len(podSpec.Volumes))
		}
		if podSpec.Volumes[0].Name != "source-code" {
			t.Errorf("volume Name = %q, want source-code", podSpec.Volumes[0].Name)
		}
		if podSpec.Volumes[0].ConfigMap == nil || podSpec.Volumes[0].ConfigMap.Name != "my-source" {
			t.Errorf("volume ConfigMap = %v, want Name=my-source", podSpec.Volumes[0].ConfigMap)
		}
		mounts := podSpec.Containers[0].VolumeMounts
		if len(mounts) != 1 {
			t.Fatalf("len(VolumeMounts) = %d, want 1", len(mounts))
		}
		if mounts[0].Name != "source-code" || mounts[0].MountPath != "/opt/code" {
			t.Errorf("VolumeMount = %+v, want Name=source-code MountPath=/opt/code", mounts[0])
		}
	})
}

func TestDeleteCronJob(t *testing.T) {
	t.Run("cronjob is deleted correctly", func(t *testing.T) {
		ctx := context.Background()
		client := fake.NewSimpleClientset(&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "delete-me",
				Namespace: "default",
			},
			Spec: batchv1.CronJobSpec{Schedule: "0 * * * *"},
		})

		if err := DeleteCronJob(ctx, client, "default", "delete-me"); err != nil {
			t.Fatalf("DeleteCronJob: %v", err)
		}

		_, err := client.BatchV1().CronJobs("default").Get(ctx, "delete-me", metav1.GetOptions{})
		if err == nil {
			t.Fatal("expected error getting deleted cronjob, got nil")
		}
		if !apierrors.IsNotFound(err) {
			t.Fatalf("expected not found error after deleting cronjob, got %v", err)
		}
	})
}
