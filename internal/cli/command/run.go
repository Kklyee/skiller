package command

import (
	"errors"
	"fmt"
	"io"

	"github.com/Kklyee/skiller/internal/agent"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/transaction"
	"github.com/spf13/cobra"
)

func NewRun() *cobra.Command {
	return newRun(agent.Run)
}

func newRun(runAgent func(string, io.Reader, io.Writer, io.Writer) error) *cobra.Command {
	return &cobra.Command{
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
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Synced %s\n", configPath); err != nil {
					return err
				}
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Launching %s\n", args[0]); err != nil {
				return err
			}
			return runAgent(args[0], cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}
