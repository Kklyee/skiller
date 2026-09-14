package command

import (
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/tui"
	"github.com/spf13/cobra"
)

func NewTUI() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive terminal interface",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			return tui.Run(pathSet)
		},
	}
}
