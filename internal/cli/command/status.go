package command

import (
	"fmt"
	"text/tabwriter"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/spf13/cobra"
)

func NewStatus() *cobra.Command {
	var asJSON bool
	command := &cobra.Command{
		Use:   "status",
		Short: "Show installed skill status",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			summary, err := catalog.SummaryFor(
				pathSet.Active,
				pathSet.Disabled,
			)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), statusJSON(summary))
			}

			w := tabwriter.NewWriter(
				cmd.OutOrStdout(),
				0,
				4,
				2,
				' ',
				0,
			)

			rows := []struct {
				label string
				value any
			}{
				{label: "Installed", value: summary.Installed},
				{label: "Active", value: summary.Active},
				{label: "Disabled", value: summary.Disabled},
				{label: "Conflict", value: summary.Conflict},
				{label: "Broken", value: summary.Broken},
				{label: "Invalid", value: summary.Invalid},
				{label: "Active dir", value: summary.ActiveDir},
				{label: "Disabled dir", value: summary.DisabledDir},
			}

			for _, row := range rows {
				if _, err := fmt.Fprintf(w, "%s\t%v\n", row.label, row.value); err != nil {
					return fmt.Errorf("write status: %w", err)
				}
			}

			if err := w.Flush(); err != nil {
				return fmt.Errorf("flush status: %w", err)
			}

			return nil
		},
	}
	command.Flags().BoolVar(&asJSON, "json", false, "write JSON output")
	return command
}

type statusJSONRow struct {
	Installed   int    `json:"installed"`
	Active      int    `json:"active"`
	Disabled    int    `json:"disabled"`
	Conflict    int    `json:"conflict"`
	Broken      int    `json:"broken"`
	Invalid     int    `json:"invalid"`
	ActiveDir   string `json:"active_dir"`
	DisabledDir string `json:"disabled_dir"`
}

func statusJSON(summary catalog.Summary) statusJSONRow {
	return statusJSONRow{
		Installed:   summary.Installed,
		Active:      summary.Active,
		Disabled:    summary.Disabled,
		Conflict:    summary.Conflict,
		Broken:      summary.Broken,
		Invalid:     summary.Invalid,
		ActiveDir:   summary.ActiveDir,
		DisabledDir: summary.DisabledDir,
	}
}
