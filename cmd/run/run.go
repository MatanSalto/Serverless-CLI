package run

import (
	"github.com/spf13/cobra"
)

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run Python code serverlessly",
	Long:  `Run Python programs on your Kubernetes cluster without managing infrastructure.`,
}

func init() {
	RunCmd.AddCommand(OneOffCmd)
	RunCmd.AddCommand(AsyncCmd)
}
