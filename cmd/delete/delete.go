package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"serverless-cli/pkg/kube"
	"serverless-cli/pkg/runner"
)

var DeleteCmd = &cobra.Command{
	Use:   "delete <workload-name>",
	Short: "Delete a workload and its source ConfigMap",
	Long:  `Delete a workload (Job, CronJob, or Service/Deployment) by name from the cluster. Removes the Kubernetes resource and the ConfigMap that stores the source code.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	if err := runner.CleanupSource(ctx, client, namespace, workloadName); err != nil {
		return err
	}

	fmt.Printf("Deleted workload %q and ConfigMap %q from namespace %q.\n", workloadName, kube.ConfigMapNameForWorkload(workloadName), namespace)
	return nil
}
