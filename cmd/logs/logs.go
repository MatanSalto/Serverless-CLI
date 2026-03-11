package logs

import (
	"context"
	"fmt"
	"os"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"

	"serverless-cli/pkg/kube"
	"serverless-cli/pkg/utils"
)

var LogsCmd = &cobra.Command{
	Use:   "logs <workload-name>",
	Short: "View logs of a workload",
	Long:  `Stream logs from a workload (Job) by name. For running jobs, logs stream until the job completes. For completed jobs, prints the existing logs and exits.`,
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
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	jobLogWriter := utils.NewJobLogsWriter(os.Stdout)
	defer jobLogWriter.Reset()

	// First, check if this is a Job.
	if _, err := client.BatchV1().Jobs(namespace).Get(ctx, workloadName, metav1.GetOptions{}); err == nil {
		if err := kube.StreamJobLogs(ctx, client, namespace, workloadName, jobLogWriter); err != nil {
			return fmt.Errorf("stream logs: %w", err)
		}
		return nil
		// Return error if the error is not "not found"
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get job: %w", err)
	}

	// If not a Job, check if it's a CronJob and stream logs from its most recent Job.
	cronJob, err := client.BatchV1().CronJobs(namespace).Get(ctx, workloadName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("workload %q not found as Job or CronJob in namespace %q", workloadName, namespace)
		}
		return fmt.Errorf("get cronjob: %w", err)
	}

	latestJobName, err := kube.GetLatestJobNameForCronJob(ctx, client, namespace, cronJob.Name)
	if err != nil {
		return fmt.Errorf("get latest job for cronjob: %w", err)
	}
	if latestJobName == "" {
		fmt.Printf("CronJob %q has not created any Jobs yet.\n", cronJob.Name)
		return nil
	}

	job, err := client.BatchV1().Jobs(namespace).Get(ctx, latestJobName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get latest job %q for cronjob %q: %w", latestJobName, cronJob.Name, err)
	}

	creationTime := job.CreationTimestamp.Time
	fmt.Printf("Streaming logs for most recent Job %q created at %s (CronJob %q).\n", latestJobName, creationTime.Format(time.RFC3339), cronJob.Name)

	if err := kube.StreamJobLogs(ctx, client, namespace, latestJobName, jobLogWriter); err != nil {
		return fmt.Errorf("stream logs for latest job: %w", err)
	}
	return nil
}
