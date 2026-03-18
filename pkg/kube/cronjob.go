package kube

import (
	"context"
	"sort"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	serr "serverless-cli/internal/errors"
)

// CronJobParams holds parameters for creating a CronJob (same as JobParams plus Schedule).
type CronJobParams struct {
	Name           string
	Namespace      string
	Schedule       string // cron expression, e.g. "0 * * * *" for hourly
	WorkloadType   string
	Image          string
	Command        []string
	Args           []string
	Env            []corev1.EnvVar
	ConfigMapName  string
	MountPath      string
	ConfigMapItems []corev1.KeyToPath
}

// CreateCronJob creates a CronJob in the cluster that runs Jobs on the given schedule.
func CreateCronJob(ctx context.Context, client kubernetes.Interface, params CronJobParams) (*batchv1.CronJob, error) {
	if params.Namespace == "" {
		return nil, serr.ValidationError{Field: "namespace", Reason: "required"}
	}
	if params.Name == "" {
		return nil, serr.ValidationError{Field: "name", Reason: "required"}
	}
	if params.Schedule == "" {
		return nil, serr.ValidationError{Field: "schedule", Reason: "required"}
	}
	if params.Image == "" {
		return nil, serr.ValidationError{Field: "image", Reason: "required"}
	}

	container := corev1.Container{
		Name:    params.Name,
		Image:   params.Image,
		Command: params.Command,
		Args:    params.Args,
		Env:     params.Env,
	}

	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers:    []corev1.Container{container},
	}

	if params.ConfigMapName != "" {
		volName := "source-code"
		podSpec.Volumes = []corev1.Volume{
			{
				Name: volName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: params.ConfigMapName},
						Items:                params.ConfigMapItems,
					},
				},
			},
		}
		mountPath := params.MountPath
		if mountPath == "" {
			mountPath = "/opt/code"
		}
		container.VolumeMounts = []corev1.VolumeMount{
			{Name: volName, MountPath: mountPath},
		}
		podSpec.Containers[0] = container
	}

	workloadType := params.WorkloadType
	if workloadType == "" {
		workloadType = WorkloadTypeCron
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: params.Namespace,
			Labels: map[string]string{
				LabelManagedKey:      LabelManagedValue,
				LabelWorkloadTypeKey: workloadType,
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule: params.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: podSpec,
					},
				},
			},
		},
	}

	return client.BatchV1().CronJobs(params.Namespace).Create(ctx, cronJob, metav1.CreateOptions{})
}

// ListManagedCronJobs returns CronJobs in the given namespace that were created by this CLI.
func ListManagedCronJobs(ctx context.Context, client kubernetes.Interface, namespace string) (*batchv1.CronJobList, error) {
	return client.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: ManagedBySelector,
	})
}

// DeleteCronJob deletes a CronJob by name in the given namespace (and its created Jobs).
func DeleteCronJob(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	propagation := metav1.DeletePropagationForeground
	return client.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
}

// GetLatestJobNameForCronJob returns the name of the most recently created Job owned by the given CronJob.
func GetLatestJobNameForCronJob(ctx context.Context, client kubernetes.Interface, namespace, cronJobName string) (string, error) {
	cronJob, err := client.BatchV1().CronJobs(namespace).Get(ctx, cronJobName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	uid := cronJob.UID

	jobList, err := client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	// get only the jobs that are owned by the cronjob
	var owned []batchv1.Job
	for _, job := range jobList.Items {
		for _, ref := range job.OwnerReferences {
			if ref.UID == uid && ref.Kind == "CronJob" {
				owned = append(owned, job)
				break
			}
		}
	}
	if len(owned) == 0 {
		return "", nil
	}

	// sort the jobs by creation timestamp, newest first
	sort.Slice(owned, func(i, j int) bool {
		return owned[j].CreationTimestamp.Before(&owned[i].CreationTimestamp)
	})
	return owned[0].Name, nil
}
