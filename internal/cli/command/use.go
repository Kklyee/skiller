package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/environment"
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
			pinned, err := loadPins(pathSet)
			if err != nil {
				return err
			}
			plan := reconcile.BuildWithPins(selected, skills, pinned)
			if err := writeActivationPlan(cmd, plan); err != nil {
				return err
			}
			if dryRun {
				return nil
			}
			if plan.HasIssues() {
				return errors.New("cannot apply plan with missing skills or catalog issues")
			}
			if plan.Changes() == 0 {
				return recordEnvironmentTarget(pathSet, environment.Target{Kind: environment.KindGroup, Name: selected.Name})
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
			if err := recordEnvironmentTarget(pathSet, environment.Target{Kind: environment.KindGroup, Name: selected.Name}); err != nil {
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
	output := cmd.OutOrStdout()
	if _, err := fmt.Fprintln(output, planContextStyle().Render("Target: "+plan.Group)); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}
	if err := writePlanSection(output, "Enable", plan.Enable, "+", planEnableStyle()); err != nil {
		return err
	}
	if err := writePlanSection(output, "Disable", plan.Disable, "-", planDisableStyle()); err != nil {
		return err
	}
	if err := writeKeepPlanSection(output, plan); err != nil {
		return err
	}
	if len(plan.Missing) > 0 {
		if err := writePlanSection(output, "Missing", plan.Missing, "?", planMissingStyle()); err != nil {
			return err
		}
	}
	if len(plan.Issues) > 0 {
		if err := writePlanSection(output, "Issues", plan.Issues, "!", planIssueStyle()); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(output, planHeadingStyle().Render("Summary")); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}
	parts := []string{
		planEnableStyle().Render(fmt.Sprintf("%d enable", len(plan.Enable))),
		planDisableStyle().Render(fmt.Sprintf("%d disable", len(plan.Disable))),
		planKeepStyle().Render(fmt.Sprintf("%d unchanged", len(plan.Keep))),
	}
	if len(plan.KeepActive)+len(plan.KeepDisabled) == len(plan.Keep) {
		parts = append(parts, planContextStyle().Render(fmt.Sprintf("(%d active, %d disabled)", len(plan.KeepActive), len(plan.KeepDisabled))))
	}
	if _, err := fmt.Fprintln(output, strings.Join(parts, "  ")); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}
	return nil
}

func writeActivationPlan(cmd *cobra.Command, plan reconcile.Plan) error {
	output := cmd.OutOrStdout()
	active := plan.FinalActive()
	disabled := plan.FinalDisabled()
	if _, err := fmt.Fprintln(output, planContextStyle().Render("Target: "+plan.Group)); err != nil {
		return fmt.Errorf("write activation plan: %w", err)
	}
	if err := writePlanSectionWithHeading(output, "Active", active, "●", planActiveKeepStyle(), planActiveKeepStyle()); err != nil {
		return err
	}
	if err := writePlanSectionWithHeading(output, "Disable", disabled, "○", planDisableStyle(), planDisableStyle()); err != nil {
		return err
	}
	if len(plan.Missing) > 0 {
		if err := writePlanSection(output, "Missing", plan.Missing, "?", planMissingStyle()); err != nil {
			return err
		}
	}
	if len(plan.Issues) > 0 {
		if err := writePlanSection(output, "Issues", plan.Issues, "!", planIssueStyle()); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(output, planHeadingStyle().Render("Final state")); err != nil {
		return fmt.Errorf("write activation plan: %w", err)
	}
	if _, err := fmt.Fprintln(output, strings.Join([]string{
		planActiveKeepStyle().Render(fmt.Sprintf("%d active", len(active))),
		planDisableStyle().Render(fmt.Sprintf("%d disable", len(disabled))),
	}, "  ")); err != nil {
		return fmt.Errorf("write activation plan: %w", err)
	}
	return nil
}

func writePlanSection(output io.Writer, title string, items []string, marker string, itemStyle lipgloss.Style) error {
	return writePlanSectionWithHeading(output, title, items, marker, planHeadingStyle(), itemStyle)
}

func writePlanSectionWithHeading(output io.Writer, title string, items []string, marker string, headingStyle lipgloss.Style, itemStyle lipgloss.Style) error {
	lines := []string{headingStyle.Render(fmt.Sprintf("%s (%d)", title, len(items)))}
	if len(items) == 0 {
		lines = append(lines, planContextStyle().Render("  (none)"))
	} else {
		for _, id := range items {
			lines = append(lines, itemStyle.Render(fmt.Sprintf("  %s %s", marker, id)))
		}
	}
	return writePlanBox(output, lines)
}

func writeKeepPlanSection(output io.Writer, plan reconcile.Plan) error {
	lines := []string{planHeadingStyle().Render(fmt.Sprintf("Keep (%d)", len(plan.Keep)))}
	if len(plan.KeepActive)+len(plan.KeepDisabled) == len(plan.Keep) && len(plan.Keep) > 0 {
		if len(plan.KeepActive) > 0 {
			lines = append(lines, planActiveKeepStyle().Render(fmt.Sprintf("  active (%d)", len(plan.KeepActive))))
			for _, id := range plan.KeepActive {
				lines = append(lines, planActiveKeepStyle().Render("    = "+id))
			}
		}
		if len(plan.KeepDisabled) > 0 {
			lines = append(lines, planDisabledKeepStyle().Render(fmt.Sprintf("  disabled (%d)", len(plan.KeepDisabled))))
			for _, id := range plan.KeepDisabled {
				lines = append(lines, planDisabledKeepStyle().Render("    = "+id))
			}
		}
	} else if len(plan.Keep) == 0 {
		lines = append(lines, planContextStyle().Render("  (none)"))
	} else {
		for _, id := range plan.Keep {
			lines = append(lines, planKeepStyle().Render("  = "+id))
		}
	}
	return writePlanBox(output, lines)
}

func writePlanBox(output io.Writer, lines []string) error {
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
	if _, err := fmt.Fprintln(output, box); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}
	return nil
}

func planHeadingStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
}

func planContextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
}

func planEnableStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
}

func planDisableStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
}

func planKeepStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
}

func planActiveKeepStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
}

func planDisabledKeepStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
}

func planMissingStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
}

func planIssueStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
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
