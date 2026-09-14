package agent

import (
	"strings"
	"testing"
)

func TestNames(t *testing.T) {
	got := Names()
	want := []string{"codex", "gemini", "opencode"}
	if len(got) != len(want) {
		t.Fatalf("names: got %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("name %d: got %q, want %q", index, got[index], want[index])
		}
	}
}

func TestValidate(t *testing.T) {
	if err := Validate("codex"); err != nil {
		t.Fatalf("validate codex: %v", err)
	}
	if err := Validate("unknown"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("validate unknown: %v", err)
	}
}
