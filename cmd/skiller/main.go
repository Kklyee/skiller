package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "skiller",
		Short: "Manage which AI skills are visible to agents",
		Long: `Skiller manages AI skill visibility.
           Active skills live in ~/.agents/skills.
           Disabled skills are parked outside the agent discovery path.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Skiller")
			fmt.Println()
			fmt.Println("AI Skill Visibility Manager")
			fmt.Println()
			fmt.Println("Run 'skiller --help' to get started.")
		},
	}

	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
