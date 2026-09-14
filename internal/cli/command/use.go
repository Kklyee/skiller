package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/Kklyee/skiller/internal/transaction"
	"github.com/spf13/cobra"
)

func NewUse() *cobra.Command {
	var dryRun bool

	command := &cobra.Command{
		Use:   "use <group>",
		Short: "Reconcile active skills to a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			selected, skills, err := useContext(pathSet, args[0])
			if err != nil {
				return err
			}
			plan := reconcile.Build(selected, skills)
			if err := writePlan(cmd, plan); err != nil {
				return err
			}
			if dryRun {
				return nil
			}
			if plan.HasIssues() {
				return errors.New("cannot apply plan with missing skills or catalog issues")
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
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Applied group %s\n", plan.Group)
			return err
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "show the plan without changing skills")

	return command
}

func useContext(pathSet paths.Set, name string) (group.Group, []catalog.Skill, error) {
	store := group.New(pathSet.Groups)
	selected, err := store.Get(name)
	if err != nil {
		return group.Group{}, nil, err
	}

	skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
	if err != nil {
		return group.Group{}, nil, fmt.Errorf("inspect installed skills: %w", err)
	}

	return selected, skills, nil
}

func writePlan(cmd *cobra.Command, plan reconcile.Plan) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Enable"); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}
	for _, id := range plan.Enable {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  + %s\n", id); err != nil {
			return fmt.Errorf("write plan: %w", err)
		}
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Disable"); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}
	for _, id := range plan.Disable {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", id); err != nil {
			return fmt.Errorf("write plan: %w", err)
		}
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Keep"); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}
	for _, id := range plan.Keep {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  = %s\n", id); err != nil {
			return fmt.Errorf("write plan: %w", err)
		}
	}
	if len(plan.Missing) > 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Missing"); err != nil {
			return fmt.Errorf("write plan: %w", err)
		}
		for _, id := range plan.Missing {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  ? %s\n", id); err != nil {
				return fmt.Errorf("write plan: %w", err)
			}
		}
	}
	if len(plan.Issues) > 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Issues"); err != nil {
			return fmt.Errorf("write plan: %w", err)
		}
		for _, issue := range plan.Issues {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  ! %s\n", issue); err != nil {
				return fmt.Errorf("write plan: %w", err)
			}
		}
	}

	_, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"%d enable  %d disable  %d unchanged\n",
		len(plan.Enable),
		len(plan.Disable),
		len(plan.Keep),
	)
	return err
}

func confirmPlan(cmd *cobra.Command) (bool, error) {
	if _, err := fmt.Fprint(cmd.OutOrStdout(), "Apply plan? [y/N]: "); err != nil {
		return false, fmt.Errorf("write confirmation: %w", err)
	}

	answer, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}
