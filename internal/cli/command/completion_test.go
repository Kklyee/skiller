package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionCommandGeneratesSupportedShells(t *testing.T) {
	markers := map[string]string{
		"bash":       "__skiller_debug",
		"fish":       "complete -c skiller",
		"powershell": "Register-ArgumentCompleter",
		"zsh":        "#compdef skiller",
	}
	for _, shell := range completionShells {
		t.Run(shell, func(t *testing.T) {
			root := &cobra.Command{Use: "skiller"}
			root.AddCommand(NewCompletion(), &cobra.Command{Use: "profile"}, &cobra.Command{Use: "group"}, &cobra.Command{Use: "use"})
			var output bytes.Buffer
			root.SetArgs([]string{"completion", shell})
			root.SetOut(&output)
			root.SetErr(&output)
			if err := root.Execute(); err != nil {
				t.Fatalf("generate %s completion: %v", shell, err)
			}
			for _, want := range []string{"skiller", markers[shell]} {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("completion missing %q:\n%s", want, output.String())
				}
			}
		})
	}
}

func TestCompletionCommandRejectsUnknownShell(t *testing.T) {
	cmd := NewCompletion()
	cmd.SetArgs([]string{"unknown"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unsupported shell") {
		t.Fatalf("unknown shell error: %v", err)
	}
}
