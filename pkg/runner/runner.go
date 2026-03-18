package runner

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	serr "serverless-cli/internal/errors"
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
	// WorkloadType sets the workload-type label on the created Job.
	// If empty, it defaults to kube.WorkloadTypeOneOff.
	WorkloadType string
}

// RunCronSourceParams holds parameters for running source code on a schedule using a CronJob.
type RunCronSourceParams struct {
	SourcePath  string   // path to source file or directory
	Namespace   string   // Kubernetes namespace
	CronJobName string   // name for the CronJob
	Schedule    string   // cron schedule (e.g. "0 * * * *")
	RunnerImage string   // container image for the runner
	Entrypoint  string   // script name to run (e.g. "main.py"), set as SLP_ENTRYPOINT
	Args        []string // optional args passed to the job
	// WorkloadType sets the workload-type label on the created CronJob.
	// If empty, it defaults to kube.WorkloadTypeCron.
	WorkloadType string
}

// RunServiceSourceParams holds parameters for running source code as a long-running service (Deployment + Service).
type RunServiceSourceParams struct {
	SourcePath    string   // path to source file or directory
	Namespace     string   // Kubernetes namespace
	ServiceName   string   // name for the Deployment and Service
	RunnerImage   string   // container image for the runner
	Entrypoint    string   // script name to run (e.g. "main.py"), set as SLP_ENTRYPOINT
	Port          int32    // container port the app listens on (e.g. 8080)
	Args          []string // optional args passed to the container
	WorkloadType  string   // if empty, defaults to kube.WorkloadTypeService
}

// RunSource packs the source into a filemap, creates a ConfigMap and a Job with a volume
// mount so the runner container sees the source at /opt/code.
func RunSource(ctx context.Context, client kubernetes.Interface, params RunSourceParams) error {
	if params.SourcePath == "" {
		return serr.ValidationError{Field: "source path", Reason: "required"}
	}
	if params.Namespace == "" {
		return serr.ValidationError{Field: "namespace", Reason: "required"}
	}
	if params.JobName == "" {
		return serr.ValidationError{Field: "job name", Reason: "required"}
	}
	if params.RunnerImage == "" {
		return serr.ValidationError{Field: "runner image", Reason: "required"}
	}
	if params.Entrypoint == "" {
		return serr.ValidationError{Field: "entrypoint", Reason: "required"}
	}

	filesMap, err := packager.BuildFileMap(params.SourcePath)
	if err != nil {
		return fmt.Errorf("build file map: %w", err)
	}

	// Check if the source total size exceeds the ConfigMap limit
	totalSize := packager.FileMapTotalSize(filesMap)
	if totalSize > ConfigMapMaxSize {
		return serr.SizeLimitError{
			Resource:    "ConfigMap",
			ActualBytes: totalSize,
			LimitBytes:  ConfigMapMaxSize,
		}
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
		return serr.KubeOpError{
			Op:        "create",
			Resource:  "ConfigMap",
			Name:      configMapName,
			Namespace: params.Namespace,
			Err:       err,
		}
	}

	// Create the runner job that runs the source code
	items := packager.FileMapToVolumeItems(filesMap)
	jobParams := kube.JobParams{
		Name:         params.JobName,
		Namespace:    params.Namespace,
		Image:        params.RunnerImage,
		MountPath:    "/opt/code",
		WorkloadType: params.WorkloadType,
		// ConfigMapName refers to the ConfigMap that holds the source code
		ConfigMapName: configMapName,
		// We pass the configmap items in order to create the volume in the job
		ConfigMapItems: items,
		Env: []corev1.EnvVar{
			{Name: "SLP_ENTRYPOINT", Value: params.Entrypoint},
		},
		Args: params.Args,
	}
	_, err = kube.CreateJob(ctx, client, jobParams)
	if err != nil {
		return serr.KubeOpError{
			Op:        "create",
			Resource:  "Job",
			Name:      params.JobName,
			Namespace: params.Namespace,
			Err:       err,
		}
	}
	return nil
}

// RunCronSource packs the source into a filemap, creates a ConfigMap and a CronJob with a volume
// mount so the runner container sees the source at /opt/code.
func RunCronSource(ctx context.Context, client kubernetes.Interface, params RunCronSourceParams) error {
	if params.SourcePath == "" {
		return serr.ValidationError{Field: "source path", Reason: "required"}
	}
	if params.Namespace == "" {
		return serr.ValidationError{Field: "namespace", Reason: "required"}
	}
	if params.CronJobName == "" {
		return serr.ValidationError{Field: "cronjob name", Reason: "required"}
	}
	if params.Schedule == "" {
		return serr.ValidationError{Field: "schedule", Reason: "required"}
	}
	if params.RunnerImage == "" {
		return serr.ValidationError{Field: "runner image", Reason: "required"}
	}
	if params.Entrypoint == "" {
		return serr.ValidationError{Field: "entrypoint", Reason: "required"}
	}

	filesMap, err := packager.BuildFileMap(params.SourcePath)
	if err != nil {
		return fmt.Errorf("build file map: %w", err)
	}

	// Check if the source total size exceeds the ConfigMap limit
	totalSize := packager.FileMapTotalSize(filesMap)
	if totalSize > ConfigMapMaxSize {
		return serr.SizeLimitError{
			Resource:    "ConfigMap",
			ActualBytes: totalSize,
			LimitBytes:  ConfigMapMaxSize,
		}
	}

	// Create the configmap for the source code
	configMapName := kube.ConfigMapNameForWorkload(params.CronJobName)
	data := packager.FileMapToConfigData(filesMap)
	_, err = kube.CreateConfigMap(ctx, client, kube.ConfigMapParams{
		Name:      configMapName,
		Namespace: params.Namespace,
		Data:      data,
	})
	if err != nil {
		return serr.KubeOpError{
			Op:        "create",
			Resource:  "ConfigMap",
			Name:      configMapName,
			Namespace: params.Namespace,
			Err:       err,
		}
	}

	// Create the runner cronjob that runs the source code on a schedule
	items := packager.FileMapToVolumeItems(filesMap)
	cronParams := kube.CronJobParams{
		Name:           params.CronJobName,
		Namespace:      params.Namespace,
		Schedule:       params.Schedule,
		WorkloadType:   params.WorkloadType,
		Image:          params.RunnerImage,
		MountPath:      "/opt/code",
		ConfigMapName:  configMapName,
		ConfigMapItems: items,
		Env: []corev1.EnvVar{
			{Name: "SLP_ENTRYPOINT", Value: params.Entrypoint},
		},
		Args: params.Args,
	}
	_, err = kube.CreateCronJob(ctx, client, cronParams)
	if err != nil {
		return serr.KubeOpError{
			Op:        "create",
			Resource:  "CronJob",
			Name:      params.CronJobName,
			Namespace: params.Namespace,
			Err:       err,
		}
	}
	return nil
}

// RunServiceSource packs the source into a filemap, creates a ConfigMap, a Deployment, and a Service
// so the runner container runs long-running and is exposed on the given port.
// It returns the created Service object.
func RunServiceSource(ctx context.Context, client kubernetes.Interface, params RunServiceSourceParams) (*corev1.Service, error) {
	if params.SourcePath == "" {
		return nil, serr.ValidationError{Field: "source path", Reason: "required"}
	}
	if params.Namespace == "" {
		return nil, serr.ValidationError{Field: "namespace", Reason: "required"}
	}
	if params.ServiceName == "" {
		return nil, serr.ValidationError{Field: "service name", Reason: "required"}
	}
	if params.RunnerImage == "" {
		return nil, serr.ValidationError{Field: "runner image", Reason: "required"}
	}
	if params.Entrypoint == "" {
		return nil, serr.ValidationError{Field: "entrypoint", Reason: "required"}
	}
	if params.Port <= 0 {
		return nil, serr.ValidationError{Field: "port", Reason: "must be positive"}
	}

	filesMap, err := packager.BuildFileMap(params.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("build file map: %w", err)
	}
	totalSize := packager.FileMapTotalSize(filesMap)
	if totalSize > ConfigMapMaxSize {
		return nil, serr.SizeLimitError{
			Resource:    "ConfigMap",
			ActualBytes: totalSize,
			LimitBytes:  ConfigMapMaxSize,
		}
	}

	configMapName := kube.ConfigMapNameForWorkload(params.ServiceName)
	data := packager.FileMapToConfigData(filesMap)
	_, err = kube.CreateConfigMap(ctx, client, kube.ConfigMapParams{
		Name:      configMapName,
		Namespace: params.Namespace,
		Data:      data,
	})
	if err != nil {
		return nil, serr.KubeOpError{
			Op:        "create",
			Resource:  "ConfigMap",
			Name:      configMapName,
			Namespace: params.Namespace,
			Err:       err,
		}
	}

	items := packager.FileMapToVolumeItems(filesMap)
	workloadType := params.WorkloadType
	if workloadType == "" {
		workloadType = kube.WorkloadTypeService
	}
	portStr := fmt.Sprintf("%d", params.Port)
	depParams := kube.DeploymentParams{
		Name:            params.ServiceName,
		Namespace:       params.Namespace,
		WorkloadType:    workloadType,
		Image:           params.RunnerImage,
		ContainerPort:   params.Port,
		MountPath:       "/opt/code",
		ConfigMapName:   configMapName,
		ConfigMapItems:  items,
		Env: []corev1.EnvVar{
			{Name: "SLP_ENTRYPOINT", Value: params.Entrypoint},
			{Name: "SLP_PORT", Value: portStr},
		},
		Args:            params.Args,
		PodLabelAppName: params.ServiceName,
	}
	_, err = kube.CreateDeployment(ctx, client, depParams)
	if err != nil {
		return nil, serr.KubeOpError{
			Op:        "create",
			Resource:  "Deployment",
			Name:      params.ServiceName,
			Namespace: params.Namespace,
			Err:       err,
		}
	}

	svcParams := kube.ServiceParams{
		Name:       params.ServiceName,
		Namespace:  params.Namespace,
		Port:       params.Port,
		TargetPort: params.Port,
		Selector:   map[string]string{"app": params.ServiceName},
		Type:       corev1.ServiceTypeNodePort,
	}
	svc, err := kube.CreateService(ctx, client, svcParams)
	if err != nil {
		return nil, serr.KubeOpError{
			Op:        "create",
			Resource:  "Service",
			Name:      params.ServiceName,
			Namespace: params.Namespace,
			Err:       err,
		}
	}
	return svc, nil
}

// CleanupSource deletes the workload and its associated source ConfigMap created by RunSource.
func CleanupSource(ctx context.Context, client kubernetes.Interface, namespace, jobName string) error {
	if namespace == "" {
		return serr.ValidationError{Field: "namespace", Reason: "required"}
	}
	if jobName == "" {
		return serr.ValidationError{Field: "job name", Reason: "required"}
	}

	// Try to delete a job first, if that doesn't exist, try to delete a CronJob with the same name
	if err := kube.DeleteJob(ctx, client, namespace, jobName); err != nil {
		if !apierrors.IsNotFound(err) {
			return serr.KubeOpError{
				Op:        "delete",
				Resource:  "Job",
				Name:      jobName,
				Namespace: namespace,
				Err:       err,
			}
		}
	}

	// Also try deleting a CronJob with this name (for cron workloads).
	if err := kube.DeleteCronJob(ctx, client, namespace, jobName); err != nil {
		if !apierrors.IsNotFound(err) {
			return serr.KubeOpError{
				Op:        "delete",
				Resource:  "CronJob",
				Name:      jobName,
				Namespace: namespace,
				Err:       err,
			}
		}
	}

	// Also try deleting a Deployment and Service with this name (for service workloads).
	if err := kube.DeleteDeployment(ctx, client, namespace, jobName); err != nil {
		if !apierrors.IsNotFound(err) {
			return serr.KubeOpError{
				Op:        "delete",
				Resource:  "Deployment",
				Name:      jobName,
				Namespace: namespace,
				Err:       err,
			}
		}
	}
	if err := kube.DeleteService(ctx, client, namespace, jobName); err != nil {
		if !apierrors.IsNotFound(err) {
			return serr.KubeOpError{
				Op:        "delete",
				Resource:  "Service",
				Name:      jobName,
				Namespace: namespace,
				Err:       err,
			}
		}
	}

	configMapName := kube.ConfigMapNameForWorkload(jobName)
	if err := kube.DeleteConfigMap(ctx, client, namespace, configMapName); err != nil {
		// if the configmap was already deleted, we don't need to return an error
		if !apierrors.IsNotFound(err) {
			return serr.KubeOpError{
				Op:        "delete",
				Resource:  "ConfigMap",
				Name:      configMapName,
				Namespace: namespace,
				Err:       err,
			}
		}
	}

	return nil
}
