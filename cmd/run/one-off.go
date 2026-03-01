package run

import (
	"github.com/spf13/cobra"
)

var OneOffCmd = &cobra.Command{
	Use:   "one-off [script] [args...]",
	Short: "Run a Python script once",
	Long:  `Run a Python script on the cluster once. The job runs to completion and exits.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runOneOff,
}

func init() {}

func runOneOff(cmd *cobra.Command, args []string) error {
	fmt.Println("Running one-off job")
	return nil
}
