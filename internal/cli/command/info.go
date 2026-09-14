package command

import (
	"fmt"
	"text/tabwriter"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/paths"
	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
	"github.com/spf13/cobra"
)

type infoLocation struct {
	location  string
	path      string
	source    string
	target    string
	skillFile string
}

func NewInfo() *cobra.Command {
	return &cobra.Command{
		Use:   "info <skill>",
		Short: "Show skill metadata and provenance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
			if err != nil {
				return fmt.Errorf("inspect installed skills: %w", err)
			}
			var selected catalog.Skill
			found := false
			for _, skill := range skills {
				if skill.ID == args[0] {
					selected = skill
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("skill %q is not installed", args[0])
			}

			metadata, ok, err := skillprovenance.New(pathSet.Provenance).Get(selected.ID)
			if err != nil {
				return err
			}
			if err := writeInfo(cmd, selected, metadata, ok); err != nil {
				return err
			}
			return nil
		},
	}
}

func writeInfo(cmd *cobra.Command, skill catalog.Skill, metadata skillprovenance.Entry, hasMetadata bool) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Skill: %s\nStatus: %s\n", skill.ID, skill.State.String()); err != nil {
		return fmt.Errorf("write skill info: %w", err)
	}

	source := "-"
	if hasMetadata && metadata.Source != "" {
		source = metadata.Source
	} else if locations := infoLocations(skill); len(locations) > 0 {
		source = locations[0].source
	}
	for _, row := range []struct {
		name  string
		value string
	}{
		{name: "Source", value: source},
		{name: "Repository", value: metadata.Repository},
		{name: "Installer", value: metadata.Installer},
		{name: "Revision", value: metadata.Revision},
	} {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", row.name, infoValue(row.value)); err != nil {
			return fmt.Errorf("write %s: %w", row.name, err)
		}
	}

	locations := infoLocations(skill)
	if len(locations) == 1 {
		row := locations[0]
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Location: %s\nPath: %s\nSource type: %s\n", row.location, row.path, row.source); err != nil {
			return fmt.Errorf("write skill location: %w", err)
		}
		if row.target != "-" {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Target: %s\n", row.target); err != nil {
				return fmt.Errorf("write skill target: %w", err)
			}
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "SKILL.md: %s\n", row.skillFile)
		return err
	}

	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Locations:"); err != nil {
		return fmt.Errorf("write skill locations: %w", err)
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "LOCATION\tSOURCE TYPE\tPATH\tTARGET\tSKILL.md"); err != nil {
		return fmt.Errorf("write location header: %w", err)
	}
	for _, row := range locations {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", row.location, row.source, row.path, row.target, row.skillFile); err != nil {
			return fmt.Errorf("write skill location: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush skill locations: %w", err)
	}
	return nil
}

func infoLocations(skill catalog.Skill) []infoLocation {
	locations := make([]infoLocation, 0, 2)
	if skill.ActivePath != "" {
		locations = append(locations, infoLocation{
			location:  "active",
			path:      skill.ActivePath,
			source:    skill.ActiveSource.String(),
			target:    infoValue(skill.ActiveLinkTarget),
			skillFile: skill.ActiveSkillFile,
		})
	}
	if skill.DisabledPath != "" {
		locations = append(locations, infoLocation{
			location:  "disabled",
			path:      skill.DisabledPath,
			source:    skill.DisabledSource.String(),
			target:    infoValue(skill.DisabledLinkTarget),
			skillFile: skill.DisabledSkillFile,
		})
	}
	return locations
}

func infoValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
