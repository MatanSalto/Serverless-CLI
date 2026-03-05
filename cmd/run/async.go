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
	asyncEntrypoint string
	asyncName       string
)

var AsyncCmd = &cobra.Command{
	Use:   "async <source-path> [args...]",
	Short: "Run a Python program asynchronously",
	Long:  `Run a Python program on the cluster asynchronously. Submits a Kubernetes Job and returns immediately without streaming logs.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAsync,
}

func init() {
	AsyncCmd.Flags().StringVar(&asyncEntrypoint, "entrypoint", "", "Script to run under /opt/code (default: main.py for dirs, or the filename for a single file)")
	AsyncCmd.Flags().StringVar(&asyncName, "name", "", "Job name (default: generated from source path)")
}

func runAsync(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	sourcePath := args[0]
	var restArgs []string
	if len(args) > 1 {
		restArgs = args[1:]
	}

	namespace, err := cmd.Root().PersistentFlags().GetString("namespace")
	if err != nil || namespace == "" {
		namespace = "serverless-workloads"
	}

	client, err := kube.NewClientSet()
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	entrypoint := asyncEntrypoint
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

	jobName := asyncName
	if jobName == "" {
		base := filepath.Base(sourcePath)
		if base == "." || base == "/" {
			base = "run"
		}
		suffix, _ := randomHex(6)
		jobName = "slp-" + base + "-" + suffix
	}

	err = runner.RunSource(ctx, client, runner.RunSourceParams{
		SourcePath:   sourcePath,
		Namespace:    namespace,
		JobName:      jobName,
		RunnerImage:  "matansalto/serverless-python:1.0.0",
		Entrypoint:   entrypoint,
		Args:         restArgs,
		WorkloadType: kube.WorkloadTypeAsync,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Async job %q created in namespace %q. Not streaming logs; check status with the list command.\n", jobName, namespace)
	return nil
}
