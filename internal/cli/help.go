package cli

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

type helpPalette struct {
	heading  lipgloss.Style
	command  lipgloss.Style
	argument lipgloss.Style
	flag     lipgloss.Style
	hint     lipgloss.Style
}

func configureHelp(command *cobra.Command) {
	command.SetHelpFunc(renderHelp)
	command.SetUsageFunc(renderUsage)
}

func newHelpPalette(_ io.Writer) helpPalette {
	return helpPalette{
		heading:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
		command:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")),
		argument: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		flag:     lipgloss.NewStyle().Foreground(lipgloss.Color("13")),
		hint:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	}
}

func renderHelp(command *cobra.Command, _ []string) {
	output := command.OutOrStdout()
	if description := strings.TrimSpace(firstNonEmpty(command.Long, command.Short)); description != "" {
		fmt.Fprintln(output, description)
	}
	if command.Runnable() || command.HasSubCommands() {
		if err := renderUsageTo(command, output); err != nil {
			fmt.Fprintln(command.ErrOrStderr(), err)
		}
	}
}

func renderUsage(command *cobra.Command) error {
	return renderUsageTo(command, command.OutOrStderr())
}

func renderUsageTo(command *cobra.Command, output io.Writer) error {
	palette := newHelpPalette(output)
	if err := writeHeading(output, palette, "Usage:"); err != nil {
		return err
	}
	if command.Runnable() {
		if _, err := fmt.Fprintf(output, "  %s\n", colorUseLine(command.UseLine(), palette)); err != nil {
			return err
		}
	}
	if command.HasAvailableSubCommands() {
		if _, err := fmt.Fprintf(output, "  %s\n", colorUseLine(command.CommandPath()+" [command]", palette)); err != nil {
			return err
		}
	}
	if len(command.Aliases) > 0 {
		if err := writeHeading(output, palette, "Aliases:"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(output, "  %s\n", palette.command.Render(command.NameAndAliases())); err != nil {
			return err
		}
	}
	if command.HasExample() {
		if err := writeHeading(output, palette, "Examples:"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(output, command.Example); err != nil {
			return err
		}
	}

	if command.HasAvailableSubCommands() {
		if err := writeAvailableCommands(output, command, palette); err != nil {
			return err
		}
	}
	if command.HasAvailableLocalFlags() {
		if err := writeFlagSection(output, "Flags:", command.LocalFlags().FlagUsages(), palette); err != nil {
			return err
		}
	}
	if command.HasAvailableInheritedFlags() {
		if err := writeFlagSection(output, "Global Flags:", command.InheritedFlags().FlagUsages(), palette); err != nil {
			return err
		}
	}
	if command.HasHelpSubCommands() {
		if err := writeHeading(output, palette, "Additional help topics:"); err != nil {
			return err
		}
		for _, child := range command.Commands() {
			if !child.IsAdditionalHelpTopicCommand() {
				continue
			}
			if _, err := fmt.Fprintf(output, "  %s %s\n", palette.command.Render(fmt.Sprintf("%-*s", command.CommandPathPadding(), child.CommandPath())), child.Short); err != nil {
				return err
			}
		}
	}
	if command.HasAvailableSubCommands() {
		if _, err := fmt.Fprintf(output, "\nUse %s for more information about a command.\n", palette.hint.Render(fmt.Sprintf("\"%s [command] --help\"", command.CommandPath()))); err != nil {
			return err
		}
	}
	return nil
}

func writeAvailableCommands(output io.Writer, command *cobra.Command, palette helpPalette) error {
	if len(command.Groups()) == 0 {
		if err := writeHeading(output, palette, "Available Commands:"); err != nil {
			return err
		}
		for _, child := range command.Commands() {
			if !child.IsAvailableCommand() && child.Name() != "help" {
				continue
			}
			if err := writeCommandLine(output, command.NamePadding(), child, palette); err != nil {
				return err
			}
		}
		return nil
	}

	for _, commandGroup := range command.Groups() {
		if err := writeHeading(output, palette, commandGroup.Title); err != nil {
			return err
		}
		for _, child := range command.Commands() {
			if child.GroupID != commandGroup.ID || (!child.IsAvailableCommand() && child.Name() != "help") {
				continue
			}
			if err := writeCommandLine(output, command.NamePadding(), child, palette); err != nil {
				return err
			}
		}
	}
	if !command.AllChildCommandsHaveGroup() {
		if err := writeHeading(output, palette, "Additional Commands:"); err != nil {
			return err
		}
		for _, child := range command.Commands() {
			if child.GroupID != "" || (!child.IsAvailableCommand() && child.Name() != "help") {
				continue
			}
			if err := writeCommandLine(output, command.NamePadding(), child, palette); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeCommandLine(output io.Writer, padding int, command *cobra.Command, palette helpPalette) error {
	_, err := fmt.Fprintf(output, "  %s %s\n", palette.command.Render(fmt.Sprintf("%-*s", padding, command.Name())), command.Short)
	return err
}

func writeFlagSection(output io.Writer, title, usages string, palette helpPalette) error {
	if err := writeHeading(output, palette, title); err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimRight(usages, " \t\r\n"), "\n") {
		if _, err := fmt.Fprintln(output, colorFlagUsage(line, palette)); err != nil {
			return err
		}
	}
	return nil
}

func writeHeading(output io.Writer, palette helpPalette, heading string) error {
	_, err := fmt.Fprintf(output, "\n%s\n", palette.heading.Render(heading))
	return err
}

func colorUseLine(line string, palette helpPalette) string {
	parts := strings.Fields(line)
	for index, part := range parts {
		if strings.HasPrefix(part, "<") || strings.HasPrefix(part, "[") {
			parts[index] = palette.argument.Render(part)
		} else {
			parts[index] = palette.command.Render(part)
		}
	}
	return strings.Join(parts, " ")
}

func colorFlagUsage(line string, palette helpPalette) string {
	trimmed := strings.TrimLeft(line, " ")
	indent := line[:len(line)-len(trimmed)]
	separator := strings.Index(trimmed, "  ")
	if separator < 0 {
		return line
	}
	name := strings.TrimRight(trimmed[:separator], " ")
	description := strings.TrimSpace(trimmed[separator:])
	return indent + palette.flag.Render(name) + "  " + description
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
