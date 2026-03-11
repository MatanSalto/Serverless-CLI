package run

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"serverless-cli/pkg/kube"
	"serverless-cli/pkg/runner"
)

var (
	cronEntrypoint string
	cronName       string
	cronSchedule   string
)

var CronCmd = &cobra.Command{
	Use:   "cron <source-path> [args...]",
	Short: "Run a Python program on a schedule",
	Long:  `Create a Kubernetes CronJob that runs your Python program on the given cron schedule.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runCron,
}

func init() {
	CronCmd.Flags().StringVar(&cronEntrypoint, "entrypoint", "", "Script to run under /opt/code (default: main.py for dirs, or the filename for a single file)")
	CronCmd.Flags().StringVar(&cronName, "name", "", "CronJob name (default: generated from source path)")
	CronCmd.Flags().StringVar(&cronSchedule, "schedule", "", "Cron schedule (e.g. \"0 * * * *\" for hourly)")
}

func runCron(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	sourcePath := args[0]
	var restArgs []string
	if len(args) > 1 {
		restArgs = args[1:]
	}

	if cronSchedule == "" {
		return fmt.Errorf("schedule is required (use --schedule)")
	}

	namespace, err := cmd.Root().PersistentFlags().GetString("namespace")
	if err != nil || namespace == "" {
		namespace = "serverless-workloads"
	}

	client, err := kube.NewClientSet()
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	entrypoint := cronEntrypoint
	if entrypoint == "" {
		abs, _ := filepath.Abs(sourcePath)
		if abs != "" {
			sourcePath = abs
		}
		info, err := os.Stat(sourcePath)
		// if the source path is a single file, use that filename as an entrypoint
		if err == nil && info != nil && !info.IsDir() {
			entrypoint = filepath.Base(sourcePath)
			// otherwise, the source path is a directory, so we use the main.py file as an entrypoint
		} else {
			entrypoint = "main.py"
		}
	}

	cronJobName := cronName
	if cronJobName == "" {
		base := filepath.Base(sourcePath)
		if base == "." || base == "/" {
			base = "run"
		}
		suffix, _ := randomHex(6)
		cronJobName = "slp-" + base + "-" + suffix
	}

	err = runner.RunCronSource(ctx, client, runner.RunCronSourceParams{
		SourcePath:   sourcePath,
		Namespace:    namespace,
		CronJobName:  cronJobName,
		Schedule:     cronSchedule,
		RunnerImage:  "matansalto/serverless-python:1.0.0",
		Entrypoint:   entrypoint,
		Args:         restArgs,
		WorkloadType: kube.WorkloadTypeCron,
	})
	if err != nil {
		return err
	}

	fmt.Printf("CronJob %q created in namespace %q on schedule %q.\n", cronJobName, namespace, cronSchedule)
	return nil
}

