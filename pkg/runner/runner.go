package runner

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	"serverless-cli/pkg/kube"
	"serverless-cli/pkg/packager"
)

const ConfigMapMaxSize = 1024 * 1024

// RunSourceParams holds parameters for running source code in the runner container.
type RunSourceParams struct {
	SourcePath  string   // path to source file or directory
	Namespace   string   // Kubernetes namespace
	JobName     string   // name for the Job
	RunnerImage string   // container image for the runner
	Entrypoint  string   // script name to run (e.g. "main.py"), set as SLP_ENTRYPOINT
	Args        []string // optional args passed to the job
}

// RunSource packs the source into a filemap, creates a ConfigMap and a Job with a volume
// mount so the runner container sees the source at /opt/code.
func RunSource(ctx context.Context, client kubernetes.Interface, params RunSourceParams) error {
	if params.SourcePath == "" {
		return errors.New("source path is required")
	}
	if params.Namespace == "" {
		return errors.New("namespace is required")
	}
	if params.JobName == "" {
		return errors.New("job name is required")
	}
	if params.RunnerImage == "" {
		return errors.New("runner image is required")
	}
	if params.Entrypoint == "" {
		return errors.New("entrypoint is required")
	}

	filesMap, err := packager.BuildFileMap(params.SourcePath)
	if err != nil {
		return fmt.Errorf("build file map: %w", err)
	}

	// Check if the source total size exceeds the ConfigMap limit
	totalSize := packager.FileMapTotalSize(filesMap)
	if totalSize > ConfigMapMaxSize {
		return fmt.Errorf("source total size %d bytes exceeds ConfigMap limit (%d bytes)", totalSize, ConfigMapMaxSize)
	}

	// Create the configmap for the source code
	configMapName := kube.ConfigMapNameForWorkload(params.JobName)
	data := packager.FileMapToConfigData(filesMap)
	_, err = kube.CreateConfigMap(ctx, client, kube.ConfigMapParams{
		Name:      configMapName,
		Namespace: params.Namespace,
		Data:      data,
	})
	if err != nil {
		return fmt.Errorf("create configmap: %w", err)
	}

	// Create the runner job that runs the source code
	items := packager.FileMapToVolumeItems(filesMap)
	jobParams := kube.JobParams{
		Name:          params.JobName,
		Namespace:     params.Namespace,
		Image:         params.RunnerImage,
		ConfigMapName: configMapName,
		MountPath:     "/opt/code",
		// We pass the configmap items in order to create the volume in the job
		ConfigMapItems: items,
		Env: []corev1.EnvVar{
			{Name: "SLP_ENTRYPOINT", Value: params.Entrypoint},
		},
		Args: params.Args,
	}
	_, err = kube.CreateJob(ctx, client, jobParams)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

// CleanupSource deletes the workload and its associated source ConfigMap created by RunSource.
func CleanupSource(ctx context.Context, client kubernetes.Interface, namespace, jobName string) error {
	if namespace == "" {
		return errors.New("namespace is required")
	}
	if jobName == "" {
		return errors.New("job name is required")
	}

	if err := kube.DeleteJob(ctx, client, namespace, jobName); err != nil {
		return fmt.Errorf("delete job: %w", err)
	}

	configMapName := kube.ConfigMapNameForWorkload(jobName)
	if err := kube.DeleteConfigMap(ctx, client, namespace, configMapName); err != nil {
		// if the configmap was already deleted, we don't need to return an error
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete configmap %q: %w", configMapName, err)
		}
	}

	return nil
}
