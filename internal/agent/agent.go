package agent

import (
	"fmt"
	"io"
	"os/exec"
	"slices"
)

var commands = map[string]string{
	"codex":    "codex",
	"gemini":   "gemini",
	"opencode": "opencode",
}

func Names() []string {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func Validate(name string) error {
	if _, ok := commands[name]; !ok {
		return fmt.Errorf("unsupported coding agent %q", name)
	}
	return nil
}

func Run(name string, in io.Reader, out, errOut io.Writer) error {
	if err := Validate(name); err != nil {
		return err
	}

	process := exec.Command(commands[name])
	process.Stdin = in
	process.Stdout = out
	process.Stderr = errOut
	if err := process.Run(); err != nil {
		return fmt.Errorf("run coding agent %q: %w", name, err)
	}
	return nil
}
