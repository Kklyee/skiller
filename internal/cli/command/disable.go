package command

import (
	"fmt"

	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/visibility"
	"github.com/spf13/cobra"
)

func NewDisable() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <skill>",
		Short: "Hide a skill from coding agents",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			id := args[0]

			changed, err := visibility.Disable(
				pathSet.Active,
				pathSet.Disabled,
				id,
			)
			if err != nil {
				return err
			}

			if !changed {
				if _, err := fmt.Fprintf(
					cmd.OutOrStdout(),
					"%s is already disabled\n",
					id,
				); err != nil {
					return fmt.Errorf("write disable result: %w", err)
				}

				return nil
			}

			if _, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"Disabled %s\n",
				id,
			); err != nil {
				return fmt.Errorf("write disable result: %w", err)
			}

			return nil
		},
	}
}
