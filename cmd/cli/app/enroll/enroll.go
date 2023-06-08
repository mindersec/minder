package enroll

import (
	"github.com/stacklok/mediator/cmd/cli/app"

	"github.com/spf13/cobra"
)

// EnrollCmd is the root command for the org subcommands
var EnrollCmd = &cobra.Command{
	Use:   "enroll",
	Short: "Manage organizations within a mediator control plane",
	Long: `The medctl org commands manage organizations within a mediator
control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("enroll called")
	},
}

func init() {
	app.RootCmd.AddCommand(EnrollCmd)
}
