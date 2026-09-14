package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/version"
)

func TestVersionCommand(t *testing.T) {
	oldVersion, oldCommit, oldDate := version.Version, version.Commit, version.Date
	t.Cleanup(func() {
		version.Version, version.Commit, version.Date = oldVersion, oldCommit, oldDate
	})
	version.Version = "1.2.3"
	version.Commit = "abc123"
	version.Date = "2026-09-14"

	var output bytes.Buffer
	cmd := NewVersion()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute version: %v", err)
	}
	for _, want := range []string{"skiller 1.2.3", "commit: abc123", "built: 2026-09-14"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("version output missing %q: %q", want, output.String())
		}
	}
}
