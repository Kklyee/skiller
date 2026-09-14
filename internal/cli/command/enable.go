package command

import (
	"fmt"

	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/visibility"
	"github.com/spf13/cobra"
)

func NewEnable() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <skill>",
		Short: "Make a skill visible to coding agents",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			id := args[0]

			changed, err := visibility.Enable(
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
					"%s is already enabled\n",
					id,
				); err != nil {
					return fmt.Errorf("write enable result: %w", err)
				}

				return nil
			}

			if _, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"Enabled %s\n",
				id,
			); err != nil {
				return fmt.Errorf("write enable result: %w", err)
			}

			return nil
		},
	}
}
