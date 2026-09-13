package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Kklyee/skiller/internal/app"
	"github.com/Kklyee/skiller/internal/discovery"
	"github.com/Kklyee/skiller/internal/storage"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := storage.DefaultPaths()
			if err != nil {
				return err
			}

			scanner := discovery.NewScanner()
			service := app.NewSkillService(paths, scanner)

			skills, err := service.List()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(
				os.Stdout,
				0,
				0,
				2,
				' ',
				0,
			)

			fmt.Fprintln(w, "SKILL\tSTATUS")

			for _, skill := range skills {
				fmt.Fprintf(
					w,
					"%s\t%s\n",
					skill.ID,
					skill.State,
				)
			}

			return w.Flush()
		},
	}
}
