package command

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/profile"
	"github.com/Kklyee/skiller/internal/project"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/Kklyee/skiller/internal/transaction"
	"github.com/spf13/cobra"
)

func NewSync() *cobra.Command {
	var dryRun bool
	var check bool

	command := &cobra.Command{
		Use:   "sync",
		Short: "Reconcile skills to the current project configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			plan, configPath, err := syncPlan(pathSet)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), planContextStyle().Render("Project config: "+configPath)); err != nil {
				return fmt.Errorf("write project config: %w", err)
			}
			if err := writePlan(cmd, plan); err != nil {
				return err
			}
			if check {
				if plan.HasIssues() {
					return checkFailure("project skill environment has issues")
				}
				if plan.Changes() > 0 {
					return checkFailure("project skill environment is out of sync")
				}
				return nil
			}
			if dryRun {
				return nil
			}
			if plan.HasIssues() {
				return errors.New("cannot apply project plan with missing skills, groups, or catalog issues")
			}
			if plan.Changes() == 0 {
				return nil
			}

			confirmed, err := confirmPlan(cmd)
			if err != nil {
				return err
			}
			if !confirmed {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
				return err
			}

			if err := transaction.Apply(pathSet, plan); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Synced %s\n", configPath)
			return err
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "show the plan without changing skills")
	command.Flags().BoolVar(&check, "check", false, "check whether the project environment is synchronized")

	return command
}

func syncPlan(pathSet paths.Set) (reconcile.Plan, string, error) {
	config, configPath, err := project.Load(".")
	if err != nil {
		return reconcile.Plan{}, "", err
	}

	skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
	if err != nil {
		return reconcile.Plan{}, "", fmt.Errorf("inspect installed skills: %w", err)
	}
	pinned, err := loadPins(pathSet)
	if err != nil {
		return reconcile.Plan{}, "", err
	}

	var selected group.Group
	var missingGroups []string
	if config.Profile != "" {
		stored, err := profile.New(pathSet.Profiles).Get(config.Profile)
		if err != nil {
			return reconcile.Plan{}, "", err
		}
		groups, err := group.New(pathSet.Groups).List()
		if err != nil {
			return reconcile.Plan{}, "", err
		}
		target := profile.Resolve(stored, groups)
		selected = project.ApplyOverrides(target.Group, config.Include, config.Exclude)
		missingGroups = target.MissingGroups
	} else {
		selected = project.ApplyOverrides(group.Group{Name: "project", Skills: config.Skills}, config.Include, config.Exclude)
	}

	plan := reconcile.BuildWithPins(selected, skills, pinned)
	for _, name := range missingGroups {
		plan.Issues = append(plan.Issues, fmt.Sprintf("missing group %s", name))
	}
	slices.Sort(plan.Issues)

	return plan, configPath, nil
}
