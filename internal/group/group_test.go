package group

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kklyee/skiller/internal/catalog"
)

func TestStoreCRUDAndMissingSkills(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "groups"))

	created, err := store.Create("coding")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if created.Name != "coding" || len(created.Skills) != 0 {
		t.Fatalf("created group: %+v", created)
	}

	added, err := store.Add("coding", "code-review", "missing", "code-review")
	if err != nil {
		t.Fatalf("add skills: %v", err)
	}
	if added != 2 {
		t.Fatalf("added: got %d, want 2", added)
	}

	groups, err := store.ListWithMissing([]catalog.Skill{{ID: "code-review"}})
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups: got %d, want 1", len(groups))
	}
	if len(groups[0].Missing) != 1 || groups[0].Missing[0] != "missing" {
		t.Fatalf("missing skills: %+v", groups[0].Missing)
	}

	removed, err := store.Remove("coding", "code-review")
	if err != nil {
		t.Fatalf("remove skill: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed: got %d, want 1", removed)
	}

	got, err := store.Get("coding")
	if err != nil {
		t.Fatalf("get group: %v", err)
	}
	if len(got.Skills) != 1 || got.Skills[0] != "missing" {
		t.Fatalf("stored skills: %+v", got.Skills)
	}

	if err := store.Delete("coding"); err != nil {
		t.Fatalf("delete group: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "coding.toml")); !os.IsNotExist(err) {
		t.Fatalf("group file still exists: %v", err)
	}
}

func TestStoreRejectsUnsafeNamesAndDuplicateFiles(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.Create("../escape"); err == nil {
		t.Fatal("expected unsafe name error")
	}

	if err := os.MkdirAll(store.Dir, 0o755); err != nil {
		t.Fatalf("create groups directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir, "coding.toml"), []byte("name = \"coding\"\nskills = [\"a\", \"a\"]\n"), 0o644); err != nil {
		t.Fatalf("write group file: %v", err)
	}
	if _, err := store.Get("coding"); err == nil {
		t.Fatal("expected duplicate skill error")
	}
}
