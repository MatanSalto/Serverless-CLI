package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"serverless-cli/cmd/list"
	"serverless-cli/cmd/run"
)

var (
	kubeconfig string
	namespace  string
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
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (if not provided, the default is $KUBECONFIG)")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "serverless-python", "The Kubernetes namespace to run workloads in")

	rootCmd.AddCommand(run.RunCmd)
	rootCmd.AddCommand(list.ListCmd)
}
