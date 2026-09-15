package command

import (
	"fmt"
	"text/tabwriter"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/spf13/cobra"
)

func NewList() *cobra.Command {
	var asJSON bool
	command := &cobra.Command{
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
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), listJSON(skills))
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
	command.Flags().BoolVar(&asJSON, "json", false, "write JSON output")
	return command
}

type listJSONRow struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func listJSON(skills []catalog.Skill) []listJSONRow {
	rows := make([]listJSONRow, 0, len(skills))
	for _, skill := range skills {
		rows = append(rows, listJSONRow{
			ID:          skill.ID,
			Status:      skill.State.String(),
			Name:        skill.Name,
			Description: skill.Description,
		})
	}
	return rows
}
