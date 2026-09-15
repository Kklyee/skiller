package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/Kklyee/skiller/internal/transaction"
	"github.com/spf13/cobra"
)

type installerRunner func(context.Context, []string, io.Reader, io.Writer, io.Writer) error

func NewInstall() *cobra.Command {
	return newInstallerCommand("install", "add", 1, runSkillsCLI)
}

func NewUpdate() *cobra.Command {
	return newInstallerCommand("update", "update", 0, runSkillsCLI)
}

func NewRemove() *cobra.Command {
	return newInstallerCommand("remove", "remove", 1, runSkillsCLI)
}

func newInstallerCommand(name, installerAction string, minimumArgs int, run installerRunner) *cobra.Command {
	command := &cobra.Command{
		Use:   name + " [skills CLI args...]",
		Short: "Bridge the skills installer",
		Args:  cobra.MinimumNArgs(minimumArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			before, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
			if err != nil {
				return fmt.Errorf("inspect skills before installer action: %w", err)
			}
			pinned, err := loadPins(pathSet)
			if err != nil {
				return err
			}

			var originalActive []string
			var temporarilyEnabled []string
			if installerAction == "update" {
				originalActive = activeSkillIDs(before)
				temporarilyEnabled, err = prepareInstallerUpdate(pathSet, before, pinned, args)
				if err != nil {
					return err
				}
			}

			installerArgs := append([]string{"skills", installerAction}, args...)
			installerErr := run(cmd.Context(), installerArgs, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
			var restoreErr error
			if installerAction == "update" && len(temporarilyEnabled) > 0 {
				restoreErr = restoreInstallerSkillEnvironment(pathSet, originalActive)
				if restoreErr == nil {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Restored disabled skill state"); err != nil {
						restoreErr = err
					}
				}
			}
			if installerErr != nil || restoreErr != nil {
				return errors.Join(installerErr, restoreErr)
			}

			after, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
			if err != nil {
				return fmt.Errorf("inspect skills after installer action: %w", err)
			}
			if conflicts := installerConflicts(after); len(conflicts) > 0 {
				return fmt.Errorf("installer left conflict skills: %s", strings.Join(conflicts, ", "))
			}
			if err := recordInstallerChanges(pathSet, installerAction, args, before, after); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Completed %s via %s\n", name, installerLabel())
			return err
		},
	}

	return command
}

func runSkillsCLI(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
	executable := os.Getenv("SKILLER_SKILLS_COMMAND")
	if executable == "" {
		executable = "npx"
	}
	command := exec.CommandContext(ctx, executable, args...)
	command.Stdin = in
	command.Stdout = out
	command.Stderr = errOut
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s: %w", installerLabel(), err)
	}
	return nil
}

func installerLabel() string {
	executable := os.Getenv("SKILLER_SKILLS_COMMAND")
	if executable == "" {
		executable = "npx"
	}
	return executable + " skills"
}

func activeSkillIDs(skills []catalog.Skill) []string {
	active := make([]string, 0, len(skills))
	for _, skill := range skills {
		if skill.State == catalog.StateActive {
			active = append(active, skill.ID)
		}
	}
	slices.Sort(active)
	return active
}

func restoreInstallerSkillEnvironment(pathSet paths.Set, originalActive []string) error {
	skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
	if err != nil {
		return fmt.Errorf("inspect skills for restore: %w", err)
	}
	pinned, err := loadPins(pathSet)
	if err != nil {
		return err
	}

	plan := reconcile.BuildWithPins(group.Group{Name: "installer-restore", Skills: originalActive}, skills, pinned)
	if plan.HasIssues() {
		return errors.New("cannot restore original skill environment with missing skills or catalog issues")
	}
	if err := transaction.Apply(pathSet, plan); err != nil {
		return fmt.Errorf("restore original skill environment: %w", err)
	}
	return nil
}

func prepareInstallerUpdate(pathSet paths.Set, skills []catalog.Skill, pinned, args []string) ([]string, error) {
	targets := installerSkillArgs(args)
	targetSet := make(map[string]struct{}, len(targets))
	for _, id := range targets {
		targetSet[id] = struct{}{}
	}

	temporarilyEnabled := make([]string, 0)
	for _, skill := range skills {
		if skill.State != catalog.StateDisabled {
			continue
		}
		if len(targetSet) > 0 {
			if _, ok := targetSet[skill.ID]; !ok {
				continue
			}
		}
		temporarilyEnabled = append(temporarilyEnabled, skill.ID)
	}

	desired := append(activeSkillIDs(skills), temporarilyEnabled...)
	plan := reconcile.BuildWithPins(group.Group{Name: "installer-update", Skills: desired}, skills, pinned)
	if plan.HasIssues() {
		return nil, errors.New("cannot prepare installer update with missing skills or catalog issues")
	}
	if err := transaction.Apply(pathSet, plan); err != nil {
		return nil, fmt.Errorf("prepare installer update: %w", err)
	}
	return temporarilyEnabled, nil
}

func installerSkillArgs(args []string) []string {
	ids := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		ids = append(ids, arg)
	}
	return ids
}

func installerConflicts(skills []catalog.Skill) []string {
	conflicts := make([]string, 0)
	for _, skill := range skills {
		if skill.State == catalog.StateConflict {
			conflicts = append(conflicts, skill.ID)
		}
	}
	return conflicts
}

func recordInstallerChanges(pathSet paths.Set, action string, args []string, before, after []catalog.Skill) error {
	store := skillprovenance.New(pathSet.Provenance)
	beforeIDs := make(map[string]struct{}, len(before))
	for _, skill := range before {
		beforeIDs[skill.ID] = struct{}{}
	}
	afterIDs := make(map[string]struct{}, len(after))
	for _, skill := range after {
		afterIDs[skill.ID] = struct{}{}
	}

	switch action {
	case "add":
		source := installerSource(args)
		for _, skill := range after {
			if _, existed := beforeIDs[skill.ID]; existed {
				continue
			}
			entry, _, err := store.Get(skill.ID)
			if err != nil {
				return err
			}
			if entry.Source == "" {
				entry.Source = source
			}
			if entry.Repository == "" {
				entry.Repository = source
			}
			if entry.Installer == "" {
				entry.Installer = installerLabel()
			}
			if err := store.Set(skill.ID, entry); err != nil {
				return fmt.Errorf("record provenance for %q: %w", skill.ID, err)
			}
		}
	case "update":
		targets := installerSkillArgs(args)
		targetSet := make(map[string]struct{}, len(targets))
		for _, id := range targets {
			targetSet[id] = struct{}{}
		}
		for _, skill := range after {
			if len(targetSet) > 0 {
				if _, ok := targetSet[skill.ID]; !ok {
					continue
				}
			}
			entry, _, err := store.Get(skill.ID)
			if err != nil {
				return err
			}
			if entry.Installer == "" {
				entry.Installer = installerLabel()
			}
			if err := store.Set(skill.ID, entry); err != nil {
				return fmt.Errorf("record provenance for %q: %w", skill.ID, err)
			}
		}
	case "remove":
		for _, skill := range before {
			if _, remains := afterIDs[skill.ID]; remains {
				continue
			}
			if err := store.Remove(skill.ID); err != nil {
				return fmt.Errorf("remove provenance for %q: %w", skill.ID, err)
			}
		}
	}
	return nil
}

func installerSource(args []string) string {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
	}
	return ""
}
