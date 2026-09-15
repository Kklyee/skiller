package project

import (
	"slices"
	"testing"

	"github.com/Kklyee/skiller/internal/group"
)

func TestApplyOverrides(t *testing.T) {
	got := ApplyOverrides(
		group.Group{Name: "backend", Skills: []string{"base", "excluded"}},
		[]string{"included", "excluded"},
		[]string{"excluded"},
	)
	if got.Name != "backend" || !slices.Equal(got.Skills, []string{"base", "included"}) {
		t.Fatalf("overrides: got %#v", got)
	}
}
