package run

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"serverless-cli/pkg/kube"
	"serverless-cli/pkg/runner"
)

var (
	oneOffEntrypoint string
	oneOffName       string
	oneOffImage      string
)

var OneOffCmd = &cobra.Command{
	Use:   "one-off <source-path> [args...]",
	Short: "Run a Python program once",
	Long:  `Run a Python program on the cluster once. Submits a Kubernetes Job that runs to completion.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runOneOff,
}

func init() {
	OneOffCmd.Flags().StringVar(&oneOffEntrypoint, "entrypoint", "", "Script to run under /opt/code (default: main.py for dirs, or the filename for a single file)")
	OneOffCmd.Flags().StringVar(&oneOffName, "name", "", "Job name (default: generated from source path)")
}

func runOneOff(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	sourcePath := args[0]
	var restArgs []string
	if len(args) > 1 {
		restArgs = args[1:]
	}

	namespace, err := cmd.Root().PersistentFlags().GetString("namespace")
	if err != nil || namespace == "" {
		namespace = "serverless-python"
	}

	client, err := kube.NewClientSet()
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	entrypoint := oneOffEntrypoint
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

	jobName := oneOffName
	if jobName == "" {
		base := filepath.Base(sourcePath)
		if base == "." || base == "/" {
			base = "run"
		}
		suffix, _ := randomHex(6)
		jobName = "slp-" + base + "-" + suffix
	}

	err = runner.RunSource(ctx, client, runner.RunSourceParams{
		SourcePath:  sourcePath,
		Namespace:   namespace,
		JobName:     jobName,
		RunnerImage: "matansalto/serverless-python:1.0.0",
		Entrypoint:  entrypoint,
		Args:        restArgs,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Job %q created in namespace %q. Source mounted at /opt/code, entrypoint %q.\n", jobName, namespace, entrypoint)
	return nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}
