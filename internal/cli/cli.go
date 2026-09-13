package cli

import (
	"github.com/Kklyee/skiller/internal/cli/command"
	"github.com/spf13/cobra"
)

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "skiller",
		Short:         "AI Skill Visibility Manager",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		command.NewList(),
		command.NewDisable(),
	)

	return cmd
}
