package repo

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ValidateRepoInput validates repo command inputs
func ValidateRepoInput(cmd *cobra.Command, requireName bool) error {
	provider, _ := cmd.Flags().GetString("provider")
	names, _ := cmd.Flags().GetStringSlice("name")
	all, _ := cmd.Flags().GetBool("all")

	command := cmd.CommandPath() // dynamic command name

	if provider == "" {
		return fmt.Errorf(`missing required flag: --provider

Example:
  %s --name owner/repo --provider github`, command)
	}

	if requireName && !all && len(names) == 0 {
		return fmt.Errorf(`missing required input.

Provide repository using:
  --name owner/repo

Or use:
  --all

Example:
  %s --name owner/repo --provider github`, command)
	}

	return nil
}
