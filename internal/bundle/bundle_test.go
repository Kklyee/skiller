package bundle

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/profile"
	"github.com/Kklyee/skiller/internal/provenance"
)

func TestBundleRoundTrip(t *testing.T) {
	want := Bundle{
		Version:  Version,
		Active:   []string{"code-review"},
		Pins:     []string{"tdd"},
		Groups:   []group.Group{{Name: "coding", Skills: []string{"code-review", "tdd"}}},
		Profiles: []profile.Profile{{Name: "backend", Groups: []string{"coding"}, Skills: []string{"research"}, Exclude: []string{"wizard"}}},
		Provenance: map[string]provenance.Entry{
			"code-review": {Source: "github", Repository: "owner/repo", Installer: "npx skills", Revision: "abc"},
		},
	}
	path := filepath.Join(t.TempDir(), "bundle.toml")
	if err := Write(path, want); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if !slices.Equal(got.Active, want.Active) || !slices.Equal(got.Pins, want.Pins) || len(got.Groups) != 1 || len(got.Profiles) != 1 || got.Provenance["code-review"] != want.Provenance["code-review"] {
		t.Fatalf("bundle: got %#v, want %#v", got, want)
	}
}

func TestBundleMissingIncludesAllReferences(t *testing.T) {
	bundle := Bundle{
		Version: Version,
		Active:  []string{"active"},
		Pins:    []string{"pinned"},
		Groups:  []group.Group{{Name: "coding", Skills: []string{"grouped"}}},
		Profiles: []profile.Profile{{
			Name:    "backend",
			Skills:  []string{"profile-skill"},
			Exclude: []string{"excluded"},
		}},
		Provenance: map[string]provenance.Entry{"source-skill": {}},
	}
	missing := bundle.Missing([]catalog.Skill{{ID: "active"}})
	want := []string{"grouped", "pinned", "profile-skill", "source-skill"}
	if !slices.Equal(missing, want) {
		t.Fatalf("missing: got %v, want %v", missing, want)
	}
}
