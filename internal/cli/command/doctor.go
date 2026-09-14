package command

import (
	"errors"
	"fmt"

	"github.com/Kklyee/skiller/internal/doctor"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/spf13/cobra"
)

func NewDoctor() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check Skiller paths and skill state",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			report := doctor.Inspect(pathSet)
			if err := writeDoctorReport(cmd, report); err != nil {
				return err
			}
			if report.Overall == doctor.Error {
				return errors.New("doctor found serious issues")
			}

			return nil
		},
	}
}

func writeDoctorReport(cmd *cobra.Command, report doctor.Report) error {
	previousSection := ""
	for _, check := range report.Checks {
		if check.Section != previousSection {
			if previousSection != "" {
				if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
					return fmt.Errorf("write doctor report: %w", err)
				}
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), check.Section); err != nil {
				return fmt.Errorf("write doctor report: %w", err)
			}
			previousSection = check.Section
		}

		if _, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"  %s %s: %s\n",
			check.Level.Symbol(),
			check.Name,
			check.Detail,
		); err != nil {
			return fmt.Errorf("write doctor report: %w", err)
		}
	}

	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"\nStatus: %s\n",
		report.Overall,
	); err != nil {
		return fmt.Errorf("write doctor status: %w", err)
	}

	return nil
}
