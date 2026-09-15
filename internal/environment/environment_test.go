package environment

import (
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.toml")
	want := Target{Kind: KindGroup, Name: "coding"}

	if err := New(path).Save(want); err != nil {
		t.Fatalf("save target: %v", err)
	}

	got, ok, err := New(path).Load()
	if err != nil {
		t.Fatalf("load target: %v", err)
	}
	if !ok {
		t.Fatal("target was not loaded")
	}
	if got != want {
		t.Fatalf("target = %+v, want %+v", got, want)
	}
}

func TestStoreMissingFileIsManual(t *testing.T) {
	target, ok, err := New(filepath.Join(t.TempDir(), "state.toml")).Load()
	if err != nil {
		t.Fatalf("load missing target: %v", err)
	}
	if ok || target != (Target{}) {
		t.Fatalf("missing target = %+v, loaded = %v", target, ok)
	}
	if got := Evaluate(false, false, 0); got != StatusManual {
		t.Fatalf("status = %s, want %s", got, StatusManual)
	}
}

func TestEvaluateTargetStatus(t *testing.T) {
	cases := []struct {
		name       string
		loaded     bool
		hasIssues  bool
		changes    int
		wantStatus Status
	}{
		{name: "manual", wantStatus: StatusManual},
		{name: "synced", loaded: true, wantStatus: StatusSynced},
		{name: "modified", loaded: true, changes: 1, wantStatus: StatusModified},
		{name: "needs attention", loaded: true, hasIssues: true, wantStatus: StatusNeedsAttention},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := Evaluate(test.loaded, test.hasIssues, test.changes); got != test.wantStatus {
				t.Fatalf("status = %s, want %s", got, test.wantStatus)
			}
		})
	}
}
