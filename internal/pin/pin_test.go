package pin

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestStoreMaintainsSortedUniquePins(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "pins.toml"))

	if got, want := mustAdd(t, store, "zeta", "alpha", "zeta"), 2; got != want {
		t.Fatalf("added pins = %d, want %d", got, want)
	}
	if got, want := mustList(t, store), []string{"alpha", "zeta"}; !slices.Equal(got, want) {
		t.Fatalf("pins = %v, want %v", got, want)
	}

	if got, want := mustRemove(t, store, "alpha", "missing"), 1; got != want {
		t.Fatalf("removed pins = %d, want %d", got, want)
	}
	if got, want := mustList(t, store), []string{"zeta"}; !slices.Equal(got, want) {
		t.Fatalf("pins after remove = %v, want %v", got, want)
	}
}

func TestStoreKeepsUnknownSkillIDsForDoctor(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "pins.toml"))
	if _, err := store.Add("not-installed"); err != nil {
		t.Fatalf("add unknown pin: %v", err)
	}
	if got, want := mustList(t, store), []string{"not-installed"}; !slices.Equal(got, want) {
		t.Fatalf("pins = %v, want %v", got, want)
	}
}

func mustAdd(t *testing.T, store Store, ids ...string) int {
	t.Helper()
	count, err := store.Add(ids...)
	if err != nil {
		t.Fatalf("add pins: %v", err)
	}
	return count
}

func mustRemove(t *testing.T, store Store, ids ...string) int {
	t.Helper()
	count, err := store.Remove(ids...)
	if err != nil {
		t.Fatalf("remove pins: %v", err)
	}
	return count
}

func mustList(t *testing.T, store Store) []string {
	t.Helper()
	pins, err := store.List()
	if err != nil {
		t.Fatalf("list pins: %v", err)
	}
	return pins
}
