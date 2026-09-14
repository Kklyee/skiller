package command

import (
	"fmt"
	"text/tabwriter"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/pin"
	"github.com/spf13/cobra"
)

func NewPin() *cobra.Command {
	return &cobra.Command{
		Use:   "pin <skills...>",
		Short: "Keep skills active in every environment",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			count, err := pin.New(pathSet.Pins).Add(args...)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Pinned %d skills\n", count)
			return err
		},
	}
}

func NewUnpin() *cobra.Command {
	return &cobra.Command{
		Use:   "unpin <skills...>",
		Short: "Remove skills from the pinned baseline",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			count, err := pin.New(pathSet.Pins).Remove(args...)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Unpinned %d skills\n", count)
			return err
		},
	}
}

func NewPins() *cobra.Command {
	return &cobra.Command{
		Use:   "pins",
		Short: "List pinned skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			pinned, err := loadPins(pathSet)
			if err != nil {
				return err
			}
			skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
			if err != nil {
				return err
			}
			byID := make(map[string]catalog.Skill, len(skills))
			for _, skill := range skills {
				byID[skill.ID] = skill
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			if _, err := fmt.Fprintln(w, "SKILL\tSTATUS"); err != nil {
				return fmt.Errorf("write pins header: %w", err)
			}
			for _, id := range pinned {
				status := "missing"
				if skill, ok := byID[id]; ok {
					status = skill.State.String()
				}
				if _, err := fmt.Fprintf(w, "%s\t%s\n", id, status); err != nil {
					return fmt.Errorf("write pin %q: %w", id, err)
				}
			}
			if err := w.Flush(); err != nil {
				return fmt.Errorf("flush pins: %w", err)
			}
			return nil
		},
	}
}

func loadPins(pathSet paths.Set) ([]string, error) {
	pinned, err := pin.New(pathSet.Pins).List()
	if err != nil {
		return nil, fmt.Errorf("read pinned skills: %w", err)
	}
	return pinned, nil
}
