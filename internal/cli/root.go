package cli

import "github.com/spf13/cobra"

func Execute() error {
	return newRootCommand().Execute()
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "skiller",
		Short:         "AI Skill Visibility Manager",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newListCommand())

	return cmd
}
