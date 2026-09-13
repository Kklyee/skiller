package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed skills",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			skills, err := catalog.Scan(
				pathSet.Active,
				pathSet.Disabled,
			)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(
				cmd.OutOrStdout(),
				0,
				4,
				2,
				' ',
				0,
			)

			if _, err := fmt.Fprintln(w, "SKILL\tSTATUS"); err != nil {
				return fmt.Errorf("write list header: %w", err)
			}

			for _, skill := range skills {
				if _, err := fmt.Fprintf(
					w,
					"%s\t%s\n",
					skill.ID,
					skill.State,
				); err != nil {
					return fmt.Errorf(
						"write skill %q: %w",
						skill.ID,
						err,
					)
				}
			}

			if err := w.Flush(); err != nil {
				return fmt.Errorf("flush skill list: %w", err)
			}

			return nil
		},
	}
}
