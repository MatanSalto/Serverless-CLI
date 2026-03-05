package kube

import (
	"context"
	"errors"
	"io"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// Job represents a Kubernetes Job.
type JobParams struct {
	// Job name
	Name string
	// Kubernetes namespace
	Namespace string
	// WorkloadType indicates the kind of workload (one-off, async, cron, website).
	// If empty, it defaults to WorkloadTypeOneOff.
	WorkloadType string
	// Container image
	Image string
	// Command to run in the container
	Command []string
	// Arguments to pass to the command
	Args []string
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
		Containers:    []corev1.Container{container},
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

	workloadType := jobParams.WorkloadType
	if workloadType == "" {
		workloadType = WorkloadTypeOneOff
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobParams.Name,
			Namespace: jobParams.Namespace,
			Labels: map[string]string{
				LabelManagedKey:      LabelManagedValue,
				LabelWorkloadTypeKey: workloadType,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}

	return client.BatchV1().Jobs(jobParams.Namespace).Create(ctx, job, metav1.CreateOptions{})
}

// WaitForJob polls until the Job completes (Succeeded or Failed) or the context is done.
func WaitForJob(ctx context.Context, client kubernetes.Interface, namespace, jobName string) error {
	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		job, err := client.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if job.Status.Succeeded >= 1 || job.Status.Failed >= 1 {
			return true, nil
		}
		return false, nil
	})
}

// GetJobLogs writes the logs of the Job's pod(s) to w. Uses the first pod found for the job.
func GetJobLogs(ctx context.Context, client kubernetes.Interface, namespace, jobName string, w io.Writer) error {
	podName, err := getJobPodName(ctx, client, namespace, jobName)
	if err != nil || podName == "" {
		return err
	}
	req := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{})
	stream, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()
	_, err = io.Copy(w, stream)
	return err
}

// getJobPodName waits for the job's pod to exist and returns its name.
func getJobPodName(ctx context.Context, client kubernetes.Interface, namespace, jobName string) (string, error) {
	var podName string
	// try running a function that checks the pod name until the cancellation of the context
	err := wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		// list only the pods that are associated with the current job
		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "job-name=" + jobName,
		})
		if err != nil {
			return false, err
		}
		if len(pods.Items) == 0 {
			return false, nil
		}
		podName = pods.Items[0].Name
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return podName, nil
}

// StreamJobLogs waits for the job's pod, then streams its logs to w in real time until the pod exits.
func StreamJobLogs(ctx context.Context, client kubernetes.Interface, namespace, jobName string, w io.Writer) error {
	podName, err := getJobPodName(ctx, client, namespace, jobName)
	if err != nil {
		return err
	}
	req := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{Follow: true})
	stream, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()
	_, err = io.Copy(w, stream)
	return err
}

// ListJobs returns all Jobs in the given namespace.
func ListJobs(ctx context.Context, client kubernetes.Interface, namespace string) (*batchv1.JobList, error) {
	return client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
}

// ListManagedJobs returns Jobs in the given namespace that were created by this CLI (have the managed label).
func ListManagedJobs(ctx context.Context, client kubernetes.Interface, namespace string) (*batchv1.JobList, error) {
	return client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: ManagedBySelector,
	})
}

// DeleteJob deletes a Job by name in the given namespace, cascading to its pods.
func DeleteJob(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	propagation := metav1.DeletePropagationForeground
	return client.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
}
