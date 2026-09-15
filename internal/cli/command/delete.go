package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/removal"
	"github.com/spf13/cobra"
)

func NewDelete() *cobra.Command {
	var skipConfirmation bool

	command := &cobra.Command{
		Use:   "delete [skills...]",
		Short: "Permanently delete installed skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			installed, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
			if err != nil {
				return fmt.Errorf("inspect installed skills: %w", err)
			}
			if len(installed) == 0 {
				return errors.New("no installed skills available to delete")
			}

			pinned, err := loadPins(pathSet)
			if err != nil {
				return err
			}
			options := deleteSkillOptions(installed, pinned)
			reader := bufio.NewReader(cmd.InOrStdin())
			selected := args
			if len(selected) == 0 {
				var cancelled bool
				selected, cancelled, err = selectSkillOptionsWithReader(cmd, "Installed skills:", options, reader)
				if err != nil {
					return err
				}
				if cancelled {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
					return err
				}
			} else if err := writeSkillOptions(cmd, options); err != nil {
				return err
			}

			if err := validateInstalledSkills(selected, installed); err != nil {
				return err
			}
			selected = uniqueSkillIDs(selected)
			if !skipConfirmation {
				confirmed, err := confirmDeleteSkills(cmd, selected, pinned, reader)
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
					return err
				}
			}

			result, err := removal.Delete(pathSet, selected...)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d skills\n", len(result.Deleted)); err != nil {
				return err
			}
			if result.PinsRemoved > 0 {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Removed %d pins\n", result.PinsRemoved); err != nil {
					return err
				}
			}
			if result.GroupsUpdated > 0 {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Updated %d groups\n", result.GroupsUpdated); err != nil {
					return err
				}
			}
			if result.ProfilesUpdated > 0 {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Updated %d profiles\n", result.ProfilesUpdated); err != nil {
					return err
				}
			}
			if result.ProvenanceRemoved > 0 {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Removed %d provenance entries\n", result.ProvenanceRemoved); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "skip the confirmation prompt")
	return command
}

func deleteSkillOptions(installed []catalog.Skill, pinned []string) []skillOption {
	pinnedSet := make(map[string]struct{}, len(pinned))
	for _, id := range pinned {
		pinnedSet[id] = struct{}{}
	}

	options := make([]skillOption, 0, len(installed))
	for _, skill := range installed {
		status := skill.State.String()
		if _, ok := pinnedSet[skill.ID]; ok {
			status += ", pinned"
		}
		options = append(options, skillOption{ID: skill.ID, Status: status})
	}
	return options
}

func writeSkillOptions(cmd *cobra.Command, options []skillOption) error {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintln(out, "Installed skills:"); err != nil {
		return fmt.Errorf("write skill picker: %w", err)
	}
	for index, option := range options {
		if _, err := fmt.Fprintf(out, "%d) %s [%s]\n", index+1, option.ID, option.Status); err != nil {
			return fmt.Errorf("write skill picker: %w", err)
		}
	}
	return nil
}

func uniqueSkillIDs(ids []string) []string {
	unique := slices.Clone(ids)
	slices.Sort(unique)
	return slices.Compact(unique)
}

func confirmDeleteSkills(cmd *cobra.Command, ids, pinned []string, reader *bufio.Reader) (bool, error) {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintln(out, "The following installed skill directories will be permanently deleted:"); err != nil {
		return false, fmt.Errorf("write deletion warning: %w", err)
	}
	for _, id := range ids {
		if _, err := fmt.Fprintf(out, "  - %s\n", id); err != nil {
			return false, fmt.Errorf("write deletion skill: %w", err)
		}
	}
	if _, err := fmt.Fprintln(out, "Pins and Group/Profile references for these skills will also be removed."); err != nil {
		return false, fmt.Errorf("write deletion warning: %w", err)
	}
	if hasPinnedSkill(ids, pinned) {
		if _, err := fmt.Fprintln(out, "Pinned skills will be unpinned because their directories are being deleted."); err != nil {
			return false, fmt.Errorf("write pinned deletion warning: %w", err)
		}
	}
	if _, err := fmt.Fprint(out, "Delete permanently? [y/N]: "); err != nil {
		return false, fmt.Errorf("write deletion confirmation: %w", err)
	}

	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read deletion confirmation: %w", err)
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

func hasPinnedSkill(ids, pinned []string) bool {
	pinnedSet := make(map[string]struct{}, len(pinned))
	for _, id := range pinned {
		pinnedSet[id] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := pinnedSet[id]; ok {
			return true
		}
	}
	return false
}
