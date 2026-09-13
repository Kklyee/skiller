package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "skiller",
		Short: "AI Skill Visibility Manager",
		Long: `Skiller controls which installed AI skills are visible
to coding agents through the shared .agents/skills convention.`,
	}

	rootCmd.AddCommand(NewListCommand())

	return rootCmd
}

func Execute() error {
	return NewRootCommand().Execute()
}
