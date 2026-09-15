package removal

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/pin"
	"github.com/Kklyee/skiller/internal/profile"
	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
)

func TestDeleteRemovesDirectoriesAndReferences(t *testing.T) {
	root := t.TempDir()
	pathSet := testPaths(root)
	createRemovalSkill(t, pathSet.Active, "alpha")
	createRemovalSkill(t, pathSet.Disabled, "beta")
	createRemovalSkill(t, pathSet.Active, "both")
	createRemovalSkill(t, pathSet.Disabled, "both")

	groups := group.New(pathSet.Groups)
	if _, err := groups.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := groups.Add("coding", "alpha", "beta", "both"); err != nil {
		t.Fatalf("add group skills: %v", err)
	}

	profiles := profile.New(pathSet.Profiles)
	if _, err := profiles.Create("backend"); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := profiles.Update("backend", profile.Profile{
		Name:    "backend",
		Skills:  []string{"alpha"},
		Exclude: []string{"beta"},
	}); err != nil {
		t.Fatalf("update profile: %v", err)
	}

	if _, err := pin.New(pathSet.Pins).Add("alpha", "both"); err != nil {
		t.Fatalf("add pins: %v", err)
	}
	provenance := skillprovenance.New(pathSet.Provenance)
	for _, id := range []string{"alpha", "beta", "both"} {
		if err := provenance.Set(id, skillprovenance.Entry{Installer: "skills"}); err != nil {
			t.Fatalf("set provenance for %s: %v", id, err)
		}
	}

	result, err := Delete(pathSet, "both", "alpha", "beta", "alpha")
	if err != nil {
		t.Fatalf("delete skills: %v", err)
	}
	if want := []string{"alpha", "beta", "both"}; !slices.Equal(result.Deleted, want) {
		t.Fatalf("deleted skills = %v, want %v", result.Deleted, want)
	}
	if result.PinsRemoved != 2 || result.GroupsUpdated != 1 || result.ProfilesUpdated != 1 || result.ProvenanceRemoved != 3 {
		t.Fatalf("cleanup result = %+v", result)
	}

	for _, id := range result.Deleted {
		for _, root := range []string{pathSet.Active, pathSet.Disabled} {
			if _, err := os.Lstat(filepath.Join(root, id)); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("deleted path %s/%s still exists: %v", root, id, err)
			}
		}
	}

	storedGroup, err := groups.Get("coding")
	if err != nil {
		t.Fatalf("read group: %v", err)
	}
	if len(storedGroup.Skills) != 0 {
		t.Fatalf("group references after delete = %v", storedGroup.Skills)
	}
	storedProfile, err := profiles.Get("backend")
	if err != nil {
		t.Fatalf("read profile: %v", err)
	}
	if len(storedProfile.Skills) != 0 || len(storedProfile.Exclude) != 0 {
		t.Fatalf("profile references after delete = %+v", storedProfile)
	}
	pins, err := pin.New(pathSet.Pins).List()
	if err != nil {
		t.Fatalf("read pins: %v", err)
	}
	if len(pins) != 0 {
		t.Fatalf("pins after delete = %v", pins)
	}
	entries, err := provenance.List()
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("provenance after delete = %v", entries)
	}
}

func TestDeleteRejectsUnknownSkillWithoutChangingInstalledSkills(t *testing.T) {
	root := t.TempDir()
	pathSet := testPaths(root)
	createRemovalSkill(t, pathSet.Active, "alpha")

	_, err := Delete(pathSet, "missing")
	if err == nil || !strings.Contains(err.Error(), `skill "missing" is not installed`) {
		t.Fatalf("unknown skill error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(pathSet.Active, "alpha")); err != nil {
		t.Fatalf("known skill changed after rejected deletion: %v", err)
	}
}

func testPaths(root string) paths.Set {
	return paths.Set{
		Active:     filepath.Join(root, "active"),
		Disabled:   filepath.Join(root, "disabled"),
		Groups:     filepath.Join(root, "groups"),
		Profiles:   filepath.Join(root, "profiles"),
		Pins:       filepath.Join(root, "pins.toml"),
		Provenance: filepath.Join(root, "provenance.toml"),
		Journal:    filepath.Join(root, "transaction.json"),
		Lock:       filepath.Join(root, "lock"),
	}
}

func createRemovalSkill(t *testing.T, parent, id string) {
	t.Helper()
	dir := filepath.Join(parent, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create skill directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# Skill\n"), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
}
