package kube

import (
	"context"
	"errors"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Job represents a Kubernetes Job.
type JobParams struct {
	Name      string
	Namespace string
	Image     string
	Command   []string
	Args      []string
}

// CreateJob creates and submits a Job to the cluster.
func CreateJob(ctx context.Context, client kubernetes.Interface, jobParams JobParams) (*batchv1.Job, error) {
	if jobParams.Namespace == "" {
		return nil, errors.New("namespace is required")
	}
	if jobParams.Name == "" {
		return nil, errors.New("name is required")
	}
	if jobParams.Image == "" {
		return nil, errors.New("image is required")
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobParams.Name,
			Namespace: jobParams.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    jobParams.Name,
							Image:   jobParams.Image,
							Command: jobParams.Command,
							Args:    jobParams.Args,
						},
					},
				},
			},
		},
	}

	return client.BatchV1().Jobs(jobParams.Namespace).Create(ctx, job, metav1.CreateOptions{})
}
