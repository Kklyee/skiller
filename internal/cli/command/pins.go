package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/pin"
	"github.com/spf13/cobra"
)

func NewPin() *cobra.Command {
	return &cobra.Command{
		Use:   "pin [skills...]",
		Short: "Keep skills active in every environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
			if err != nil {
				return fmt.Errorf("inspect installed skills: %w", err)
			}

			selected := args
			if len(selected) == 0 {
				pinned, err := loadPins(pathSet)
				if err != nil {
					return err
				}
				available := unpinnedSkills(skills, pinned)
				var cancelled bool
				selected, cancelled, err = selectSkills(cmd, available)
				if err != nil {
					return err
				}
				if cancelled {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
					return err
				}
			}

			if err := validateInstalledSkills(selected, skills); err != nil {
				return err
			}

			count, err := pin.New(pathSet.Pins).Add(selected...)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Pinned %d skills\n", count)
			return err
		},
	}
}

func unpinnedSkills(installed []catalog.Skill, pinned []string) []catalog.Skill {
	pinnedSet := make(map[string]struct{}, len(pinned))
	for _, id := range pinned {
		pinnedSet[id] = struct{}{}
	}

	available := make([]catalog.Skill, 0, len(installed))
	for _, skill := range installed {
		if _, ok := pinnedSet[skill.ID]; ok {
			continue
		}
		available = append(available, skill)
	}
	return available
}

func validateInstalledSkills(selected []string, installed []catalog.Skill) error {
	known := make(map[string]struct{}, len(installed))
	for _, skill := range installed {
		known[skill.ID] = struct{}{}
	}

	missing := make([]string, 0)
	for _, id := range selected {
		if _, ok := known[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if len(missing) == 1 {
		return fmt.Errorf("skill %q is not installed", missing[0])
	}
	return fmt.Errorf("skills are not installed: %s", strings.Join(missing, ", "))
}

func selectSkills(cmd *cobra.Command, installed []catalog.Skill) ([]string, bool, error) {
	if len(installed) == 0 {
		return nil, false, errors.New("no unpinned skills available to pin")
	}

	options := make([]skillOption, 0, len(installed))
	for _, skill := range installed {
		options = append(options, skillOption{ID: skill.ID, Status: skill.State.String()})
	}
	return selectSkillOptions(cmd, "Available skills to pin:", options)
}

type skillOption struct {
	ID     string
	Status string
}

func selectPinnedSkills(cmd *cobra.Command, pinned []string, installed []catalog.Skill) ([]string, bool, error) {
	if len(pinned) == 0 {
		return nil, false, errors.New("no pinned skills available to unpin")
	}

	statusByID := make(map[string]string, len(installed))
	for _, skill := range installed {
		statusByID[skill.ID] = skill.State.String()
	}

	options := make([]skillOption, 0, len(pinned))
	for _, id := range pinned {
		status := "missing"
		if value, ok := statusByID[id]; ok {
			status = value
		}
		options = append(options, skillOption{ID: id, Status: status})
	}
	return selectSkillOptions(cmd, "Pinned skills:", options)
}

func selectSkillOptions(cmd *cobra.Command, title string, options []skillOption) ([]string, bool, error) {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintln(out, title); err != nil {
		return nil, false, fmt.Errorf("write skill picker: %w", err)
	}
	for index, option := range options {
		if _, err := fmt.Fprintf(out, "%d) %s [%s]\n", index+1, option.ID, option.Status); err != nil {
			return nil, false, fmt.Errorf("write skill picker: %w", err)
		}
	}
	if _, err := fmt.Fprintln(out, "Select numbers separated by commas, e.g. 1,3,5 (a=all, q=cancel):"); err != nil {
		return nil, false, fmt.Errorf("write skill picker prompt: %w", err)
	}
	if _, err := fmt.Fprint(out, "> "); err != nil {
		return nil, false, fmt.Errorf("write skill picker prompt: %w", err)
	}

	selection, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, fmt.Errorf("read skill selection: %w", err)
	}
	selection = strings.TrimSpace(selection)
	if selection == "" || strings.EqualFold(selection, "q") || strings.EqualFold(selection, "esc") {
		return nil, true, nil
	}
	if strings.EqualFold(selection, "a") || strings.EqualFold(selection, "all") {
		selected := make([]string, 0, len(options))
		for _, option := range options {
			selected = append(selected, option.ID)
		}
		return selected, false, nil
	}

	selected := make([]string, 0)
	seen := make(map[int]struct{})
	for _, token := range strings.FieldsFunc(selection, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	}) {
		index, err := strconv.Atoi(token)
		if err != nil || index < 1 || index > len(options) {
			return nil, false, fmt.Errorf("invalid skill selection %q: choose numbers from 1 to %d", token, len(options))
		}
		if _, ok := seen[index]; ok {
			continue
		}
		seen[index] = struct{}{}
		selected = append(selected, options[index-1].ID)
	}
	if len(selected) == 0 {
		return nil, false, errors.New("select at least one skill")
	}
	return selected, false, nil
}

func NewUnpin() *cobra.Command {
	return &cobra.Command{
		Use:   "unpin [skills...]",
		Short: "Remove skills from the pinned baseline",
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			selected := args
			if len(selected) == 0 {
				pinned, err := loadPins(pathSet)
				if err != nil {
					return err
				}
				skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
				if err != nil {
					return fmt.Errorf("inspect installed skills: %w", err)
				}
				var cancelled bool
				selected, cancelled, err = selectPinnedSkills(cmd, pinned, skills)
				if err != nil {
					return err
				}
				if cancelled {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
					return err
				}
			}

			count, err := pin.New(pathSet.Pins).Remove(selected...)
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
