package command

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Kklyee/skiller/internal/bundle"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/pin"
	"github.com/Kklyee/skiller/internal/profile"
	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/Kklyee/skiller/internal/transaction"
	"github.com/spf13/cobra"
)

func NewExport() *cobra.Command {
	return &cobra.Command{
		Use:   "export <file>",
		Short: "Export the skill environment configuration",
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
			groups, err := group.New(pathSet.Groups).List()
			if err != nil {
				return err
			}
			profiles, err := profile.New(pathSet.Profiles).List()
			if err != nil {
				return err
			}
			pins, err := loadPins(pathSet)
			if err != nil {
				return err
			}
			provenanceData, err := skillprovenance.New(pathSet.Provenance).List()
			if err != nil {
				return err
			}

			stored := bundle.Bundle{
				Version:    bundle.Version,
				Active:     activeSkillIDs(skills),
				Pins:       pins,
				Groups:     groups,
				Profiles:   profiles,
				Provenance: provenanceData,
			}
			if err := bundle.Write(args[0], stored); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Exported skill environment to %s\n", args[0])
			return err
		},
	}
}

func NewImport() *cobra.Command {
	return newImportCommand(runSkillsCLI)
}

func newImportCommand(run installerRunner) *cobra.Command {
	var installMissing bool
	var replace bool
	var yes bool

	command := &cobra.Command{
		Use:   "import <file>",
		Short: "Import a skill environment configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			stored, err := bundle.Load(args[0])
			if err != nil {
				return err
			}
			installed, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
			if err != nil {
				return fmt.Errorf("inspect installed skills: %w", err)
			}
			missing := stored.Missing(installed)
			if len(missing) > 0 && !installMissing {
				return fmt.Errorf("bundle references missing skills: %s (use --install-missing)", strings.Join(missing, ", "))
			}
			if len(missing) > 0 {
				if err := installBundleMissing(cmd, stored, missing, run); err != nil {
					return err
				}
				installed, err = catalog.Scan(pathSet.Active, pathSet.Disabled)
				if err != nil {
					return fmt.Errorf("inspect skills after installation: %w", err)
				}
				if missing = stored.Missing(installed); len(missing) > 0 {
					return fmt.Errorf("skills still missing after installation: %s", strings.Join(missing, ", "))
				}
			}

			if err := checkImportTargets(pathSet, stored, replace); err != nil {
				return err
			}
			plan := reconcile.BuildWithPins(group.Group{Name: "import", Skills: stored.Active}, installed, stored.Pins)
			if err := writePlan(cmd, plan); err != nil {
				return err
			}
			if plan.HasIssues() {
				return errors.New("cannot import environment with catalog issues")
			}
			if plan.Changes() > 0 && !yes {
				confirmed, err := confirmPlan(cmd)
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
					return err
				}
			}
			if plan.Changes() > 0 {
				if err := transaction.Apply(pathSet, plan); err != nil {
					return err
				}
			}
			if err := applyBundleMetadata(pathSet, stored, replace); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Imported skill environment from %s\n", args[0])
			return err
		},
	}
	command.Flags().BoolVar(&installMissing, "install-missing", false, "install missing skills from bundle provenance")
	command.Flags().BoolVar(&replace, "replace", false, "replace existing groups, profiles, pins, and provenance")
	command.Flags().BoolVarP(&yes, "yes", "y", false, "apply the environment without confirmation")

	return command
}

func installBundleMissing(cmd *cobra.Command, stored bundle.Bundle, missing []string, run installerRunner) error {
	for _, id := range missing {
		entry, ok := stored.Provenance[id]
		if !ok {
			return fmt.Errorf("missing skill %q has no repository or source in bundle", id)
		}
		source := entry.Repository
		if source == "" {
			source = entry.Source
		}
		if source == "" {
			return fmt.Errorf("missing skill %q has no repository or source in bundle", id)
		}
		args := []string{"skills", "add", source, "--skill", id, "-y"}
		if err := run(cmd.Context(), args, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
			return fmt.Errorf("install missing skill %q: %w", id, err)
		}
	}
	return nil
}

func checkImportTargets(pathSet paths.Set, stored bundle.Bundle, replace bool) error {
	groups := group.New(pathSet.Groups)
	for _, wanted := range stored.Groups {
		current, err := groups.Get(wanted.Name)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				continue
			}
			return err
		}
		if !replace && !slices.Equal(current.Skills, wanted.Skills) {
			return fmt.Errorf("group %q already exists; use --replace to import over it", wanted.Name)
		}
	}

	profiles := profile.New(pathSet.Profiles)
	for _, wanted := range stored.Profiles {
		current, err := profiles.Get(wanted.Name)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				continue
			}
			return err
		}
		if !replace && !sameProfile(current, wanted) {
			return fmt.Errorf("profile %q already exists; use --replace to import over it", wanted.Name)
		}
	}

	if !replace {
		currentProvenance, err := skillprovenance.New(pathSet.Provenance).List()
		if err != nil {
			return err
		}
		for id, wanted := range stored.Provenance {
			if current, ok := currentProvenance[id]; ok && current != wanted {
				return fmt.Errorf("provenance for %q already exists with different values; use --replace to import over it", id)
			}
		}
	}
	return nil
}

func applyBundleMetadata(pathSet paths.Set, stored bundle.Bundle, replace bool) error {
	groups := group.New(pathSet.Groups)
	for _, wanted := range stored.Groups {
		if !replace {
			if _, err := groups.Get(wanted.Name); err == nil {
				continue
			}
		}
		if err := groups.Save(wanted, replace); err != nil {
			return err
		}
	}

	profiles := profile.New(pathSet.Profiles)
	for _, wanted := range stored.Profiles {
		if !replace {
			if _, err := profiles.Get(wanted.Name); err == nil {
				continue
			}
		}
		if err := profiles.Save(wanted, replace); err != nil {
			return err
		}
	}

	if replace {
		if err := pin.New(pathSet.Pins).Replace(stored.Pins); err != nil {
			return err
		}
		if err := skillprovenance.New(pathSet.Provenance).Replace(stored.Provenance); err != nil {
			return err
		}
	} else {
		if _, err := pin.New(pathSet.Pins).Add(stored.Pins...); err != nil {
			return err
		}
		store := skillprovenance.New(pathSet.Provenance)
		for id, entry := range stored.Provenance {
			if err := store.Set(id, entry); err != nil {
				return err
			}
		}
	}
	return nil
}

func sameProfile(a, b profile.Profile) bool {
	return slices.Equal(a.Groups, b.Groups) && slices.Equal(a.Skills, b.Skills) && slices.Equal(a.Exclude, b.Exclude)
}
