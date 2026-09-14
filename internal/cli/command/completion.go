package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

var completionShells = []string{"bash", "fish", "powershell", "zsh"}

func NewCompletion() *cobra.Command {
	return &cobra.Command{
		Use:       "completion <shell>",
		Short:     "Generate shell completion script",
		Args:      cobra.ExactArgs(1),
		ValidArgs: completionShells,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			switch args[0] {
			case "bash":
				err = cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), true)
			case "zsh":
				err = cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				err = cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				err = cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell %q (supported: bash, fish, powershell, zsh)", args[0])
			}
			if err != nil {
				return fmt.Errorf("generate %s completion: %w", args[0], err)
			}
			return nil
		},
	}
}
