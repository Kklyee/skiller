package command

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/spf13/cobra"
)

func NewGroup() *cobra.Command {
	command := &cobra.Command{
		Use:   "group",
		Short: "Manage skill groups",
		Args:  cobra.NoArgs,
	}

	command.AddCommand(
		newGroupList(),
		newGroupCreate(),
		newGroupDelete(),
		newGroupShow(),
		newGroupAdd(),
		newGroupRemove(),
	)

	return command
}

func newGroupList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List groups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, skills, err := groupContext()
			if err != nil {
				return err
			}

			groups, err := store.ListWithMissing(skills)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			if _, err := fmt.Fprintln(w, "GROUP\tSKILLS\tMISSING"); err != nil {
				return fmt.Errorf("write group header: %w", err)
			}
			for _, group := range groups {
				missing := "-"
				if len(group.Missing) > 0 {
					missing = strings.Join(group.Missing, ",")
				}
				if _, err := fmt.Fprintf(w, "%s\t%d\t%s\n", group.Name, len(group.Skills), missing); err != nil {
					return fmt.Errorf("write group %q: %w", group.Name, err)
				}
			}
			if err := w.Flush(); err != nil {
				return fmt.Errorf("flush groups: %w", err)
			}

			return nil
		},
	}
}

func newGroupCreate() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, _, err := groupContext()
			if err != nil {
				return err
			}
			if _, err := store.Create(args[0]); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created group %s\n", args[0])
			return err
		},
	}
}

func newGroupDelete() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, _, err := groupContext()
			if err != nil {
				return err
			}
			if err := store.Delete(args[0]); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Deleted group %s\n", args[0])
			return err
		},
	}
}

func newGroupShow() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, skills, err := groupContext()
			if err != nil {
				return err
			}
			storedGroup, err := store.Get(args[0])
			if err != nil {
				return err
			}
			storedGroup.Missing = group.MissingSkills(storedGroup, skills)

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\nSkills:\n", storedGroup.Name); err != nil {
				return fmt.Errorf("write group: %w", err)
			}
			for _, skill := range storedGroup.Skills {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", skill); err != nil {
					return fmt.Errorf("write group skill: %w", err)
				}
			}
			if len(storedGroup.Missing) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "Missing: none")
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Missing:"); err != nil {
				return fmt.Errorf("write missing skills: %w", err)
			}
			for _, skill := range storedGroup.Missing {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", skill); err != nil {
					return fmt.Errorf("write missing skill: %w", err)
				}
			}

			return nil
		},
	}
}

func newGroupAdd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <group> <skills...>",
		Short: "Add skills to a group",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, _, err := groupContext()
			if err != nil {
				return err
			}
			count, err := store.Add(args[0], args[1:]...)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Added %d skills to %s\n", count, args[0])
			return err
		},
	}
}

func newGroupRemove() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <group> <skills...>",
		Short: "Remove skills from a group",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, _, err := groupContext()
			if err != nil {
				return err
			}
			count, err := store.Remove(args[0], args[1:]...)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Removed %d skills from %s\n", count, args[0])
			return err
		},
	}
}

func groupContext() (group.Store, []catalog.Skill, error) {
	pathSet, err := paths.Default()
	if err != nil {
		return group.Store{}, nil, err
	}

	skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
	if err != nil {
		return group.Store{}, nil, fmt.Errorf("inspect installed skills: %w", err)
	}

	return group.New(pathSet.Groups), skills, nil
}
