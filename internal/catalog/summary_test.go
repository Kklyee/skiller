package catalog

import "testing"

func TestSummarize(t *testing.T) {
	summary := Summarize([]Skill{
		{ID: "active", State: StateActive},
		{ID: "disabled", State: StateDisabled},
		{ID: "conflict", State: StateConflict},
		{ID: "broken", State: StateBroken},
		{ID: "invalid", State: StateInvalid},
	})

	if summary.Installed != 5 {
		t.Fatalf("installed: got %d, want 5", summary.Installed)
	}
	if summary.Active != 1 {
		t.Fatalf("active: got %d, want 1", summary.Active)
	}
	if summary.Disabled != 1 {
		t.Fatalf("disabled: got %d, want 1", summary.Disabled)
	}
	if summary.Conflict != 1 {
		t.Fatalf("conflict: got %d, want 1", summary.Conflict)
	}
	if summary.Broken != 1 {
		t.Fatalf("broken: got %d, want 1", summary.Broken)
	}
	if summary.Invalid != 1 {
		t.Fatalf("invalid: got %d, want 1", summary.Invalid)
	}
}
