package catalog

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

type diskSkill struct {
	ID   string
	Path string
}

func Scan(activeDir, disabledDir string) ([]Skill, error) {
	active, err := scanDir(activeDir)
	if err != nil {
		return nil, fmt.Errorf("scan active skills: %w", err)
	}

	disabled, err := scanDir(disabledDir)
	if err != nil {
		return nil, fmt.Errorf("scan disabled skills: %w", err)
	}

	byID := make(map[string]Skill, len(active)+len(disabled))

	for _, entry := range active {
		skill := byID[entry.ID]
		skill.ID = entry.ID
		skill.ActivePath = entry.Path

		byID[entry.ID] = skill
	}

	for _, entry := range disabled {
		skill := byID[entry.ID]
		skill.ID = entry.ID
		skill.DisabledPath = entry.Path

		byID[entry.ID] = skill
	}

	skills := make([]Skill, 0, len(byID))

	for _, skill := range byID {
		switch {
		case skill.ActivePath != "" && skill.DisabledPath != "":
			skill.State = StateConflict

		case skill.ActivePath != "":
			skill.State = StateActive

		case skill.DisabledPath != "":
			skill.State = StateDisabled
		}

		skills = append(skills, skill)
	}

	slices.SortFunc(skills, func(a, b Skill) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return skills, nil
}

func scanDir(dir string) ([]diskSkill, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read directory %q: %w", dir, err)
	}

	skills := make([]diskSkill, 0, len(entries))

	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())

		info, err := os.Stat(entryPath)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", entryPath, err)
		}

		if !info.IsDir() {
			continue
		}

		skillFile := filepath.Join(entryPath, "SKILL.md")

		info, err = os.Stat(skillFile)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", skillFile, err)
		}

		if !info.Mode().IsRegular() {
			continue
		}

		skills = append(skills, diskSkill{
			ID:   entry.Name(),
			Path: entryPath,
		})
	}

	return skills, nil
}
