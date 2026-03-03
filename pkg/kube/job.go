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
	// Job name
	Name      string
	// Kubernetes namespace
	Namespace string
	// Container image
	Image     string
	// Command to run in the container
	Command   []string
	// Arguments to pass to the command
	Args      []string
	// Environment variables to set in the container
	Env []corev1.EnvVar
	// Name of the ConfigMap to mount as a volume
	ConfigMapName string
	// Path in the container at which to mount the ConfigMap
	MountPath string
	// Items to mount from the ConfigMap
	ConfigMapItems []corev1.KeyToPath
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

	container := corev1.Container{
		Name:    jobParams.Name,
		Image:   jobParams.Image,
		Command: jobParams.Command,
		Args:    jobParams.Args,
		Env:     jobParams.Env,
	}

	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers:   []corev1.Container{container},
	}

	if jobParams.ConfigMapName != "" {
		volName := "source-code"
		podSpec.Volumes = []corev1.Volume{
			{
				Name: volName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: jobParams.ConfigMapName},
						Items:                jobParams.ConfigMapItems,
					},
				},
			},
		}
		mountPath := jobParams.MountPath
		if mountPath == "" {
			mountPath = "/opt/code"
		}
		container.VolumeMounts = []corev1.VolumeMount{
			{Name: volName, MountPath: mountPath},
		}
		podSpec.Containers[0] = container
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobParams.Name,
			Namespace: jobParams.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}

	return client.BatchV1().Jobs(jobParams.Namespace).Create(ctx, job, metav1.CreateOptions{})
}
