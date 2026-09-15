package project

import (
	"slices"

	"github.com/Kklyee/skiller/internal/group"
)

func ApplyOverrides(base group.Group, include, exclude []string) group.Group {
	desired := make(map[string]struct{}, len(base.Skills)+len(include))
	for _, id := range base.Skills {
		desired[id] = struct{}{}
	}
	for _, id := range include {
		desired[id] = struct{}{}
	}
	for _, id := range exclude {
		delete(desired, id)
	}

	selected := make([]string, 0, len(desired))
	for id := range desired {
		selected = append(selected, id)
	}
	slices.Sort(selected)
	return group.Group{Name: base.Name, Skills: selected}
}
