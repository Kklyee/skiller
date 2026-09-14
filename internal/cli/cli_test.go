package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCommandRejectsUnfinishedJournal(t *testing.T) {
	root := t.TempDir()
	journal := filepath.Join(root, "transaction.json")
	if err := os.WriteFile(journal, []byte("unfinished"), 0o600); err != nil {
		t.Fatalf("create journal: %v", err)
	}

	t.Setenv("SKILLER_ACTIVE_DIR", filepath.Join(root, "active"))
	t.Setenv("SKILLER_DISABLED_DIR", filepath.Join(root, "disabled"))
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", journal)

	var output bytes.Buffer
	command := NewRootCommand()
	command.SetArgs([]string{"list"})
	command.SetOut(&output)
	command.SetErr(&output)

	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "unfinished transaction journal") {
		t.Fatalf("unexpected journal error: %v", err)
	}
}
