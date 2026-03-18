package kube

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	serr "serverless-cli/internal/errors"
)

// DeploymentParams holds parameters for creating a Deployment (long-running service workload).
type DeploymentParams struct {
	Name            string
	Namespace       string
	WorkloadType    string
	Image           string
	Command         []string
	Args            []string
	Env             []corev1.EnvVar
	ContainerPort   int32
	ConfigMapName   string
	MountPath       string
	ConfigMapItems  []corev1.KeyToPath
	Replicas        int32
	PodLabelAppName string // label value for app=, used by Service selector; defaults to Name
}

// CreateDeployment creates a Deployment in the cluster (e.g. for a service workload).
func CreateDeployment(ctx context.Context, client kubernetes.Interface, params DeploymentParams) (*appsv1.Deployment, error) {
	if params.Namespace == "" {
		return nil, serr.ValidationError{Field: "namespace", Reason: "required"}
	}
	if params.Name == "" {
		return nil, serr.ValidationError{Field: "name", Reason: "required"}
	}
	if params.Image == "" {
		return nil, serr.ValidationError{Field: "image", Reason: "required"}
	}

	appLabel := params.PodLabelAppName
	if appLabel == "" {
		appLabel = params.Name
	}

	container := corev1.Container{
		Name:    params.Name,
		Image:   params.Image,
		Command: params.Command,
		Args:    params.Args,
		Env:     params.Env,
	}
	if params.ContainerPort > 0 {
		container.Ports = []corev1.ContainerPort{
			{ContainerPort: params.ContainerPort, Protocol: corev1.ProtocolTCP},
		}
	}

	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyAlways,
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
		workloadType = WorkloadTypeService
	}

	replicas := params.Replicas
	if replicas <= 0 {
		replicas = 1
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: params.Namespace,
			Labels: map[string]string{
				LabelManagedKey:      LabelManagedValue,
				LabelWorkloadTypeKey: workloadType,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": appLabel},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						LabelManagedKey:      LabelManagedValue,
						LabelWorkloadTypeKey: workloadType,
						"app":                appLabel,
					},
				},
				Spec: podSpec,
			},
		},
	}

	return client.AppsV1().Deployments(params.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
}

// ListManagedDeployments returns Deployments in the given namespace that were created by this CLI.
func ListManagedDeployments(ctx context.Context, client kubernetes.Interface, namespace string) (*appsv1.DeploymentList, error) {
	return client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: ManagedBySelector,
	})
}

// DeleteDeployment deletes a Deployment by name in the given namespace (cascading to its pods).
func DeleteDeployment(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	propagation := metav1.DeletePropagationForeground
	return client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
}

// GetFirstRunningPodNameForDeployment returns the name of the first Running pod for the given Deployment.
func GetFirstRunningPodNameForDeployment(ctx context.Context, client kubernetes.Interface, namespace, deploymentName string) (string, error) {
	dep, err := client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return "", err
	}
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return "", err
	}
	for _, p := range pods.Items {
		// return the first running pod
		if p.Status.Phase == corev1.PodRunning {
			return p.Name, nil
		}
	}
	return "", nil
}
