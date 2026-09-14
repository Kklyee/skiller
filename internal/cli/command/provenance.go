package command

import (
	"fmt"
	"text/tabwriter"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/spf13/cobra"
)

type provenanceRow struct {
	skill     string
	state     string
	location  string
	source    string
	path      string
	target    string
	skillFile string
}

func NewProvenance() *cobra.Command {
	return &cobra.Command{
		Use:   "provenance <skill...>",
		Short: "Show where installed skills come from",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
			if err != nil {
				return fmt.Errorf("inspect installed skills: %w", err)
			}

			byID := make(map[string]catalog.Skill, len(skills))
			for _, skill := range skills {
				byID[skill.ID] = skill
			}

			rows := make([]provenanceRow, 0, len(args))
			for _, id := range args {
				skill, ok := byID[id]
				if !ok {
					return fmt.Errorf("skill %q is not installed", id)
				}
				rows = append(rows, provenanceRows(skill)...)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			if _, err := fmt.Fprintln(w, "SKILL\tSTATE\tLOCATION\tSOURCE\tPATH\tTARGET\tSKILL.md"); err != nil {
				return fmt.Errorf("write provenance header: %w", err)
			}
			for _, row := range rows {
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", row.skill, row.state, row.location, row.source, row.path, row.target, row.skillFile); err != nil {
					return fmt.Errorf("write provenance for %q: %w", row.skill, err)
				}
			}
			if err := w.Flush(); err != nil {
				return fmt.Errorf("flush provenance: %w", err)
			}
			return nil
		},
	}
}

func provenanceRows(skill catalog.Skill) []provenanceRow {
	rows := make([]provenanceRow, 0, 2)
	if skill.ActivePath != "" {
		rows = append(rows, provenanceRow{
			skill:     skill.ID,
			state:     skill.State.String(),
			location:  "active",
			source:    skill.ActiveSource.String(),
			path:      skill.ActivePath,
			target:    provenanceValue(skill.ActiveLinkTarget),
			skillFile: skill.ActiveSkillFile,
		})
	}
	if skill.DisabledPath != "" {
		rows = append(rows, provenanceRow{
			skill:     skill.ID,
			state:     skill.State.String(),
			location:  "disabled",
			source:    skill.DisabledSource.String(),
			path:      skill.DisabledPath,
			target:    provenanceValue(skill.DisabledLinkTarget),
			skillFile: skill.DisabledSkillFile,
		})
	}
	return rows
}

func provenanceValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
