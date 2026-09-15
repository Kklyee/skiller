package command

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/environment"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/profile"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/Kklyee/skiller/internal/transaction"
	"github.com/spf13/cobra"
)

func NewProfile() *cobra.Command {
	command := &cobra.Command{
		Use:   "profile",
		Short: "Manage skill profiles",
		Args:  cobra.NoArgs,
	}

	command.AddCommand(
		newProfileList(),
		newProfileShow(),
		newProfileCreate(),
		newProfileEdit(),
		newProfileDelete(),
		newProfileUse(),
	)

	return command
}

func newProfileList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			store := profile.New(pathSet.Profiles)
			profiles, err := store.List()
			if err != nil {
				return err
			}
			groups, skills, err := profileInspectionContext(pathSet)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			if _, err := fmt.Fprintln(w, "PROFILE\tGROUPS\tSKILLS\tEXCLUDE\tMISSING"); err != nil {
				return fmt.Errorf("write profile header: %w", err)
			}
			for _, stored := range profiles {
				target := profile.Resolve(stored, groups)
				missing := profileMissing(target, skills)
				if _, err := fmt.Fprintf(
					w,
					"%s\t%d\t%d\t%d\t%s\n",
					stored.Name,
					len(stored.Groups),
					len(stored.Skills),
					len(stored.Exclude),
					missing,
				); err != nil {
					return fmt.Errorf("write profile %q: %w", stored.Name, err)
				}
			}
			if err := w.Flush(); err != nil {
				return fmt.Errorf("flush profiles: %w", err)
			}

			return nil
		},
	}
}

func newProfileShow() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			stored, err := profile.New(pathSet.Profiles).Get(args[0])
			if err != nil {
				return err
			}
			groups, skills, err := profileInspectionContext(pathSet)
			if err != nil {
				return err
			}
			target := profile.Resolve(stored, groups)

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", stored.Name); err != nil {
				return fmt.Errorf("write profile name: %w", err)
			}
			if err := writeNamedList(cmd, "Groups", stored.Groups); err != nil {
				return err
			}
			if err := writeNamedList(cmd, "Skills", stored.Skills); err != nil {
				return err
			}
			if err := writeNamedList(cmd, "Exclude", stored.Exclude); err != nil {
				return err
			}
			if err := writeNamedList(cmd, "Missing groups", target.MissingGroups); err != nil {
				return err
			}
			if err := writeNamedList(cmd, "Missing skills", profile.MissingSkills(target, skills)); err != nil {
				return err
			}

			return nil
		},
	}
}

func newProfileCreate() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			if _, err := profile.New(pathSet.Profiles).Create(args[0]); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created profile %s\n", args[0])
			return err
		},
	}
}

func newProfileEdit() *cobra.Command {
	var groups, skills, exclude []string

	command := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			store := profile.New(pathSet.Profiles)
			stored, err := store.Get(args[0])
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("groups") {
				stored.Groups = emptyFlagList(groups)
			}
			if cmd.Flags().Changed("skills") {
				stored.Skills = emptyFlagList(skills)
			}
			if cmd.Flags().Changed("exclude") {
				stored.Exclude = emptyFlagList(exclude)
			}

			if err := store.Update(args[0], stored); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated profile %s\n", args[0])
			return err
		},
	}
	command.Flags().StringSliceVar(&groups, "groups", nil, "comma-separated group names")
	command.Flags().StringSliceVar(&skills, "skills", nil, "comma-separated skill IDs")
	command.Flags().StringSliceVar(&exclude, "exclude", nil, "comma-separated skill IDs to exclude")

	return command
}

func newProfileDelete() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			if err := profile.New(pathSet.Profiles).Delete(args[0]); err != nil {
				return err
			}
			if err := clearEnvironmentTarget(pathSet, environment.KindProfile, args[0], ""); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Deleted profile %s\n", args[0])
			return err
		},
	}
}

func newProfileUse() *cobra.Command {
	var dryRun bool

	command := &cobra.Command{
		Use:   "use <name>",
		Short: "Reconcile active skills to a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			stored, groups, skills, err := profileUseContext(pathSet, args[0])
			if err != nil {
				return err
			}
			target := profile.Resolve(stored, groups)
			pinned, err := loadPins(pathSet)
			if err != nil {
				return err
			}
			plan := reconcile.BuildWithPins(target.Group, skills, pinned)
			for _, name := range target.MissingGroups {
				plan.Issues = append(plan.Issues, fmt.Sprintf("missing group %s", name))
			}
			slices.Sort(plan.Issues)

			if err := writePlan(cmd, plan); err != nil {
				return err
			}
			if dryRun {
				return nil
			}
			if plan.HasIssues() {
				return errors.New("cannot apply profile with missing groups, skills, or catalog issues")
			}
			if plan.Changes() == 0 {
				return recordEnvironmentTarget(pathSet, environment.Target{Kind: environment.KindProfile, Name: stored.Name})
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
			if err := recordEnvironmentTarget(pathSet, environment.Target{Kind: environment.KindProfile, Name: stored.Name}); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Applied profile %s\n", stored.Name)
			return err
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "show the plan without changing skills")

	return command
}

func profileInspectionContext(pathSet paths.Set) ([]group.Group, []catalog.Skill, error) {
	groups, err := group.New(pathSet.Groups).List()
	if err != nil {
		return nil, nil, err
	}
	skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect installed skills: %w", err)
	}
	return groups, skills, nil
}

func profileUseContext(pathSet paths.Set, name string) (profile.Profile, []group.Group, []catalog.Skill, error) {
	stored, err := profile.New(pathSet.Profiles).Get(name)
	if err != nil {
		return profile.Profile{}, nil, nil, err
	}
	groups, skills, err := profileInspectionContext(pathSet)
	if err != nil {
		return profile.Profile{}, nil, nil, err
	}
	return stored, groups, skills, nil
}

func profileMissing(target profile.Target, installed []catalog.Skill) string {
	missing := make([]string, 0, len(target.MissingGroups))
	for _, name := range target.MissingGroups {
		missing = append(missing, "group:"+name)
	}
	for _, id := range profile.MissingSkills(target, installed) {
		missing = append(missing, "skill:"+id)
	}
	if len(missing) == 0 {
		return "-"
	}
	return strings.Join(missing, ",")
}

func writeNamedList(cmd *cobra.Command, name string, values []string) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", name); err != nil {
		return fmt.Errorf("write profile %s: %w", strings.ToLower(name), err)
	}
	if len(values) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "  none")
		return err
	}
	for _, value := range values {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", value); err != nil {
			return fmt.Errorf("write profile %s: %w", strings.ToLower(name), err)
		}
	}
	return nil
}

func emptyFlagList(values []string) []string {
	if len(values) == 1 && strings.TrimSpace(values[0]) == "" {
		return []string{}
	}
	return values
}
