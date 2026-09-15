package command

import (
	"fmt"

	"github.com/Kklyee/skiller/internal/doctor"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/spf13/cobra"
)

func NewDoctor() *cobra.Command {
	var asJSON bool
	command := &cobra.Command{
		Use:   "doctor",
		Short: "Check Skiller paths and skill state",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			pathSet, err := paths.Default()
			if err != nil {
				return err
			}

			report := doctor.Inspect(pathSet)
			if asJSON {
				if err := writeDoctorJSON(cmd, report); err != nil {
					return err
				}
				if report.Overall == doctor.Error {
					return checkFailure("doctor found serious issues")
				}
				return nil
			}
			if err := writeDoctorReport(cmd, report); err != nil {
				return err
			}
			if report.Overall == doctor.Error {
				return checkFailure("doctor found serious issues")
			}

			return nil
		},
	}
	command.Flags().BoolVar(&asJSON, "json", false, "write JSON output")
	return command
}

type doctorJSONCheck struct {
	Section string `json:"section"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Detail  string `json:"detail"`
}

type doctorJSONReport struct {
	Status  string            `json:"status"`
	Checks  []doctorJSONCheck `json:"checks"`
	Summary statusJSONRow     `json:"summary"`
}

func writeDoctorJSON(cmd *cobra.Command, report doctor.Report) error {
	checks := make([]doctorJSONCheck, 0, len(report.Checks))
	for _, check := range report.Checks {
		checks = append(checks, doctorJSONCheck{
			Section: check.Section,
			Name:    check.Name,
			Status:  check.Level.String(),
			Detail:  check.Detail,
		})
	}
	return writeJSON(cmd.OutOrStdout(), doctorJSONReport{
		Status:  report.Overall.String(),
		Checks:  checks,
		Summary: statusJSON(report.Summary),
	})
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
