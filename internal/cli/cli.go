package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/Kklyee/skiller/internal/cli/command"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/spf13/cobra"
)

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "skiller",
		Short:         "AI Skill Visibility Manager",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "doctor" {
				return nil
			}

			pathSet, err := paths.Default()
			if err != nil {
				return err
			}
			if _, err := os.Lstat(pathSet.Journal); err == nil {
				return fmt.Errorf("unfinished transaction journal exists at %s", pathSet.Journal)
			} else if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("inspect transaction journal %q: %w", pathSet.Journal, err)
			}

			return nil
		},
	}

	cmd.AddCommand(
		command.NewList(),
		command.NewDisable(),
		command.NewEnable(),
		command.NewStatus(),
		command.NewDoctor(),
		command.NewGroup(),
		command.NewUse(),
		command.NewTUI(),
	)

	return cmd
}
