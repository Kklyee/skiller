package provenance

import (
	"path/filepath"
	"testing"
)

func TestStoreSetGetAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "provenance.toml")
	store := New(path)
	want := Entry{
		Source:     "github",
		Repository: "owner/repo",
		Installer:  "skills",
		Revision:   "abc123",
	}
	if err := store.Set("code-review", want); err != nil {
		t.Fatalf("set provenance: %v", err)
	}
	got, ok, err := store.Get("code-review")
	if err != nil {
		t.Fatalf("get provenance: %v", err)
	}
	if !ok || got != want {
		t.Fatalf("provenance: got %#v, %t, want %#v, true", got, ok, want)
	}
	if err := store.Remove("code-review"); err != nil {
		t.Fatalf("remove provenance: %v", err)
	}
	if _, ok, err := store.Get("code-review"); err != nil || ok {
		t.Fatalf("removed provenance: ok=%t err=%v", ok, err)
	}
}

func TestStoreMissingFileIsEmpty(t *testing.T) {
	entries, err := New(filepath.Join(t.TempDir(), "missing.toml")).List()
	if err != nil {
		t.Fatalf("list missing provenance: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries: got %d, want 0", len(entries))
	}
}
