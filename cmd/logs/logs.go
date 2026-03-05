package logs

import (
	"context"
	"fmt"
	"os"

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

	if err := kube.StreamJobLogs(ctx, client, namespace, workloadName, jobLogWriter); err != nil {
		return fmt.Errorf("stream logs: %w", err)
	}
	return nil
}
