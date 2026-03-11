package list

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"serverless-cli/pkg/kube"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workloads",
	Long:  `List all serverless Python workloads in the configured namespace.`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	namespace, err := cmd.Root().PersistentFlags().GetString("namespace")
	if err != nil || namespace == "" {
		namespace = "serverless-workloads"
	}

	client, err := kube.NewClientSet()
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	jobList, err := kube.ListManagedJobs(ctx, client, namespace)
	if err != nil {
		return fmt.Errorf("list jobs: %w", err)
	}

	cronJobList, err := kube.ListManagedCronJobs(ctx, client, namespace)
	if err != nil {
		return fmt.Errorf("list cronjobs: %w", err)
	}

	deployList, err := kube.ListManagedDeployments(ctx, client, namespace)
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}

	if len(jobList.Items) == 0 && len(cronJobList.Items) == 0 && len(deployList.Items) == 0 {
		fmt.Printf("No workloads in namespace %q.\n", namespace)
		return nil
	}

	fmt.Printf("%-36s %-10s %-12s %s\n", "NAME", "TYPE", "STATUS", "AGE")
	fmt.Println("--------------------------------------------------------------------------------")

	// List Jobs (one-off, async, etc.)
	for _, job := range jobList.Items {
		workloadType := job.Labels[kube.LabelWorkloadTypeKey]
		if workloadType == "" {
			workloadType = kube.WorkloadTypeOneOff
		}
		status := "Running"
		if job.Status.Succeeded >= 1 {
			status = "Succeeded"
		} else if job.Status.Failed >= 1 {
			status = "Failed"
		}
		age := "—"
		if !job.CreationTimestamp.IsZero() {
			age = formatDuration(job.CreationTimestamp.Time)
		}
		fmt.Printf("%-36s %-10s %-12s %s\n", job.Name, workloadType, status, age)
	}

	// List CronJobs
	for _, cj := range cronJobList.Items {
		workloadType := cj.Labels[kube.LabelWorkloadTypeKey]
		if workloadType == "" {
			workloadType = kube.WorkloadTypeCron
		}
		status := "Scheduled"
		if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
			status = "Suspended"
		}
		age := "—"
		if !cj.CreationTimestamp.IsZero() {
			age = formatDuration(cj.CreationTimestamp.Time)
		}
		fmt.Printf("%-36s %-10s %-12s %s\n", cj.Name, workloadType, status, age)
	}

	// List Deployments (services)
	for _, dep := range deployList.Items {
		workloadType := dep.Labels[kube.LabelWorkloadTypeKey]
		if workloadType == "" {
			workloadType = kube.WorkloadTypeService
		}
		status := "Running"
		if dep.Status.ReadyReplicas < 1 && dep.Status.UpdatedReplicas < 1 {
			status = "Pending"
		}
		age := "—"
		if !dep.CreationTimestamp.IsZero() {
			age = formatDuration(dep.CreationTimestamp.Time)
		}
		fmt.Printf("%-36s %-10s %-12s %s\n", dep.Name, workloadType, status, age)
	}
	return nil
}

// formatDuration formats a duration as a string in the format of "1h2m3s"
// for example, 1 hour, 2 minutes, 3 seconds will be formatted as "1h2m3s"
func formatDuration(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
