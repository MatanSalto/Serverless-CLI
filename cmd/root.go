package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"serverless-cli/cmd/delete"
	"serverless-cli/cmd/list"
	"serverless-cli/cmd/logs"
	"serverless-cli/cmd/run"
)

var (
	namespace string
)

var rootCmd = &cobra.Command{
	Use:   "serverless-cli",
	Short: "Run Python workloads serverlessly on Kubernetes",
	Long: `A CLI that runs your Python code on your Kubernetes cluster
without managing infrastructure.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "serverless-workloads", "The Kubernetes namespace to run workloads in")

	rootCmd.AddCommand(run.RunCmd)
	rootCmd.AddCommand(list.ListCmd)
	rootCmd.AddCommand(logs.LogsCmd)
	rootCmd.AddCommand(delete.DeleteCmd)
}
