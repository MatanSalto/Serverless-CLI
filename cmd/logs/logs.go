package logs

import (
	"context"
	"fmt"
	"os"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"

	serr "serverless-cli/internal/errors"
	"serverless-cli/pkg/kube"
	"serverless-cli/pkg/utils"
)

var LogsCmd = &cobra.Command{
	Use:   "logs <workload-name>",
	Short: "View logs of a workload",
	Long:  `Stream logs from a workload (Job, CronJob, or Service/Deployment) by name. For jobs, logs stream until the job completes. For services, streams logs from one of the deployment's pods.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runLogs,
}

func runLogs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	workloadName := args[0]

	namespace, err := cmd.Root().PersistentFlags().GetString("namespace")
	if err != nil || namespace == "" {
		namespace = "serverless-workloads"
	}

	client, err := kube.NewClientSet()
	if err != nil {
		return serr.KubeOpError{
			Op:       "create",
			Resource: "kubernetes client",
			Err:      err,
		}
	}

	jobLogWriter := utils.NewJobLogsWriter(os.Stdout)
	defer jobLogWriter.Reset()

	// First, check if this is a Job.
	if _, err := client.BatchV1().Jobs(namespace).Get(ctx, workloadName, metav1.GetOptions{}); err == nil {
		if err := kube.StreamJobLogs(ctx, client, namespace, workloadName, jobLogWriter); err != nil {
			return serr.KubeOpError{
				Op:        "stream",
				Resource:  "Job logs",
				Name:      workloadName,
				Namespace: namespace,
				Err:       err,
			}
		}
		return nil
		// Return error if the error is not "not found"
	} else if !apierrors.IsNotFound(err) {
		return serr.KubeOpError{
			Op:        "get",
			Resource:  "Job",
			Name:      workloadName,
			Namespace: namespace,
			Err:       err,
		}
	}

	// If not a Job, check if it's a Deployment (service) and stream logs from one of its pods.
	if _, err := client.AppsV1().Deployments(namespace).Get(ctx, workloadName, metav1.GetOptions{}); err == nil {
		podName, err := kube.GetFirstRunningPodNameForDeployment(ctx, client, namespace, workloadName)
		if err != nil {
			return serr.KubeOpError{
				Op:        "get",
				Resource:  "Pod for Deployment",
				Name:      workloadName,
				Namespace: namespace,
				Err:       err,
			}
		}
		if podName == "" {
			return serr.StateError{
				Resource:  "Deployment",
				Name:      workloadName,
				Namespace: namespace,
				Reason:    "has no running pods yet",
			}
		}
		if err := kube.StreamPodLogs(ctx, client, namespace, podName, jobLogWriter); err != nil {
			return serr.KubeOpError{
				Op:        "stream",
				Resource:  "Pod logs",
				Name:      podName,
				Namespace: namespace,
				Err:       err,
			}
		}
		return nil
	} else if !apierrors.IsNotFound(err) {
		return serr.KubeOpError{
			Op:        "get",
			Resource:  "Deployment",
			Name:      workloadName,
			Namespace: namespace,
			Err:       err,
		}
	}

	// If not a Deployment, check if it's a CronJob and stream logs from its most recent Job.
	cronJob, err := client.BatchV1().CronJobs(namespace).Get(ctx, workloadName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return serr.NotFoundError{
				Resource:  "Workload",
				Name:      workloadName,
				Namespace: namespace,
			}
		}
		return serr.KubeOpError{
			Op:        "get",
			Resource:  "CronJob",
			Name:      workloadName,
			Namespace: namespace,
			Err:       err,
		}
	}

	latestJobName, err := kube.GetLatestJobNameForCronJob(ctx, client, namespace, cronJob.Name)
	if err != nil {
		return serr.KubeOpError{
			Op:        "get",
			Resource:  "latest Job for CronJob",
			Name:      cronJob.Name,
			Namespace: namespace,
			Err:       err,
		}
	}
	if latestJobName == "" {
		fmt.Printf("CronJob %q has not created any Jobs yet.\n", cronJob.Name)
		return nil
	}

	job, err := client.BatchV1().Jobs(namespace).Get(ctx, latestJobName, metav1.GetOptions{})
	if err != nil {
		return serr.KubeOpError{
			Op:        "get",
			Resource:  "Job",
			Name:      latestJobName,
			Namespace: namespace,
			Err:       err,
		}
	}

	creationTime := job.CreationTimestamp.Time
	fmt.Printf("Streaming logs for most recent Job %q created at %s (CronJob %q).\n", latestJobName, creationTime.Format(time.RFC3339), cronJob.Name)

	if err := kube.StreamJobLogs(ctx, client, namespace, latestJobName, jobLogWriter); err != nil {
		return serr.KubeOpError{
			Op:        "stream",
			Resource:  "Job logs",
			Name:      latestJobName,
			Namespace: namespace,
			Err:       err,
		}
	}
	return nil
}
