package profile

import (
	"path/filepath"
	"testing"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
)

func TestStoreCRUD(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "profiles"))

	created, err := store.Create("backend")
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if created.Name != "backend" {
		t.Fatalf("created profile: %#v", created)
	}

	updated := Profile{
		Name:    "backend",
		Groups:  []string{"coding", "services"},
		Skills:  []string{"research"},
		Exclude: []string{"frontend"},
	}
	if err := store.Update("backend", updated); err != nil {
		t.Fatalf("update profile: %v", err)
	}

	got, err := store.Get("backend")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if got.Name != updated.Name || !sameStrings(got.Groups, updated.Groups) || !sameStrings(got.Skills, updated.Skills) || !sameStrings(got.Exclude, updated.Exclude) {
		t.Fatalf("profile: got %#v, want %#v", got, updated)
	}

	profiles, err := store.List()
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "backend" {
		t.Fatalf("profiles: %#v", profiles)
	}

	if err := store.Delete("backend"); err != nil {
		t.Fatalf("delete profile: %v", err)
	}
	if _, err := store.Get("backend"); err == nil {
		t.Fatal("get deleted profile succeeded")
	}
}

func TestResolveCombinesGroupsSkillsAndExclusions(t *testing.T) {
	target := Resolve(
		Profile{
			Name:    "go-backend",
			Groups:  []string{"coding", "missing"},
			Skills:  []string{"research", "frontend"},
			Exclude: []string{"frontend", "tdd"},
		},
		[]group.Group{{Name: "coding", Skills: []string{"code-review", "tdd"}}},
	)

	if target.Group.Name != "go-backend" {
		t.Fatalf("target group name: %q", target.Group.Name)
	}
	if !sameStrings(target.Group.Skills, []string{"code-review", "research"}) {
		t.Fatalf("target skills: %v", target.Group.Skills)
	}
	if !sameStrings(target.MissingGroups, []string{"missing"}) {
		t.Fatalf("missing groups: %v", target.MissingGroups)
	}

	missing := MissingSkills(target, []catalog.Skill{{ID: "research", State: catalog.StateActive}})
	if !sameStrings(missing, []string{"code-review"}) {
		t.Fatalf("missing skills: %v", missing)
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
