package command

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/Kklyee/skiller/internal/agent"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/environment"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/Kklyee/skiller/internal/transaction"
	"github.com/spf13/cobra"
)

func NewRun() *cobra.Command {
	return newRun(agent.Run)
}

func newRun(runAgent func(string, io.Reader, io.Writer, io.Writer) error) *cobra.Command {
	var restore bool
	command := &cobra.Command{
		Use:       "run <agent>",
		Short:     "Synchronize skills and launch a coding agent",
		Args:      cobra.ExactArgs(1),
		ValidArgs: agent.Names(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := agent.Validate(args[0]); err != nil {
				return err
			}

			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			var originalActive []string
			if restore {
				skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
				if err != nil {
					return fmt.Errorf("snapshot active skills: %w", err)
				}
				originalActive = activeSkillIDs(skills)
			}

			plan, configPath, err := syncPlan(pathSet)
			if err != nil {
				return err
			}
			if err := writePlan(cmd, plan); err != nil {
				return err
			}
			if plan.HasIssues() {
				return errors.New("cannot run agent with missing skills, groups, or catalog issues")
			}
			if plan.Changes() > 0 {
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
				if err := recordEnvironmentTarget(pathSet, environment.Target{Kind: environment.KindProject, Name: "project", Path: configPath}); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Synced %s\n", configPath); err != nil {
					return err
				}
			} else if err := recordEnvironmentTarget(pathSet, environment.Target{Kind: environment.KindProject, Name: "project", Path: configPath}); err != nil {
				return err
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Launching %s\n", args[0]); err != nil {
				return err
			}

			agentErr := runAgent(args[0], cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
			if !restore {
				return agentErr
			}

			restoreErr := restoreSkillEnvironment(pathSet, originalActive)
			if restoreErr == nil {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Restored original skill environment"); err != nil {
					restoreErr = err
				}
			}
			return errors.Join(agentErr, restoreErr)
		},
	}
	command.Flags().BoolVar(&restore, "restore", false, "restore the original skill environment after the agent exits")

	return command
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

func restoreSkillEnvironment(pathSet paths.Set, originalActive []string) error {
	skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
	if err != nil {
		return fmt.Errorf("inspect skills for restore: %w", err)
	}
	pinned, err := loadPins(pathSet)
	if err != nil {
		return err
	}

	plan := reconcile.BuildWithPins(group.Group{Name: "restore", Skills: originalActive}, skills, pinned)
	if plan.HasIssues() {
		return errors.New("cannot restore original skill environment with missing skills or catalog issues")
	}
	if err := transaction.Apply(pathSet, plan); err != nil {
		return fmt.Errorf("restore original skill environment: %w", err)
	}
	return nil
}
