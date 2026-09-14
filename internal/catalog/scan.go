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
	ID          string
	Path        string
	State       State
	Source      Source
	LinkTarget  string
	Issue       string
	Name        string
	Description string
	SkillFile   string
}

type observedSkill struct {
	skill         Skill
	activeState   State
	disabledState State
	activeName    string
	activeDesc    string
	disabledName  string
	disabledDesc  string
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

	byID := make(map[string]observedSkill, len(active)+len(disabled))

	for _, entry := range active {
		observed := byID[entry.ID]
		observed.skill.ID = entry.ID
		observed.skill.ActivePath = entry.Path
		observed.skill.ActiveSource = entry.Source
		observed.skill.ActiveLinkTarget = entry.LinkTarget
		observed.skill.ActiveIssue = entry.Issue
		observed.skill.ActiveSkillFile = entry.SkillFile
		observed.activeState = entry.State
		observed.activeName = entry.Name
		observed.activeDesc = entry.Description

		byID[entry.ID] = observed
	}

	for _, entry := range disabled {
		observed := byID[entry.ID]
		observed.skill.ID = entry.ID
		observed.skill.DisabledPath = entry.Path
		observed.skill.DisabledSource = entry.Source
		observed.skill.DisabledLinkTarget = entry.LinkTarget
		observed.skill.DisabledIssue = entry.Issue
		observed.skill.DisabledSkillFile = entry.SkillFile
		observed.disabledState = entry.State
		observed.disabledName = entry.Name
		observed.disabledDesc = entry.Description

		byID[entry.ID] = observed
	}

	skills := make([]Skill, 0, len(byID))

	for _, observed := range byID {
		skill := observed.skill

		switch {
		case skill.ActivePath != "" && skill.DisabledPath != "":
			skill.State = StateConflict

		case skill.ActivePath != "":
			skill.State = observed.activeState

		case skill.DisabledPath != "":
			switch observed.disabledState {
			case StateBroken:
				skill.State = StateBroken
			case StateInvalid:
				skill.State = StateInvalid
			default:
				skill.State = StateDisabled
			}
		}

		if skill.ActivePath != "" {
			skill.Name = observed.activeName
			skill.Description = observed.activeDesc
			skill.SkillFile = skill.ActiveSkillFile
		} else {
			skill.Name = observed.disabledName
			skill.Description = observed.disabledDesc
			skill.SkillFile = skill.DisabledSkillFile
		}
		if skill.Name == "" {
			skill.Name = skill.ID
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

		info, err := os.Lstat(entryPath)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", entryPath, err)
		}

		source := SourceDirectory
		linkTarget := ""
		isLink := false

		if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			source, linkTarget, isLink = linkSource(entryPath)
		}

		if isLink {
			info, err = os.Stat(entryPath)
			if errors.Is(err, fs.ErrNotExist) {
				skills = append(skills, diskSkill{
					ID:         entry.Name(),
					Path:       entryPath,
					State:      StateBroken,
					Source:     source,
					LinkTarget: linkTarget,
					Issue:      "link target does not exist",
					SkillFile:  filepath.Join(entryPath, "SKILL.md"),
				})
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("stat link target %q: %w", entryPath, err)
			}
		} else if !info.IsDir() {
			continue
		}

		if !info.IsDir() {
			skills = append(skills, diskSkill{
				ID:         entry.Name(),
				Path:       entryPath,
				State:      StateInvalid,
				Source:     source,
				LinkTarget: linkTarget,
				Issue:      "skill path is not a directory",
				SkillFile:  filepath.Join(entryPath, "SKILL.md"),
			})
			continue
		}

		skillFile := filepath.Join(entryPath, "SKILL.md")

		info, err = os.Stat(skillFile)
		if errors.Is(err, fs.ErrNotExist) {
			skills = append(skills, diskSkill{
				ID:         entry.Name(),
				Path:       entryPath,
				State:      StateInvalid,
				Source:     source,
				LinkTarget: linkTarget,
				Issue:      "missing SKILL.md",
				SkillFile:  skillFile,
			})
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", skillFile, err)
		}

		if !info.Mode().IsRegular() {
			skills = append(skills, diskSkill{
				ID:         entry.Name(),
				Path:       entryPath,
				State:      StateInvalid,
				Source:     source,
				LinkTarget: linkTarget,
				Issue:      "SKILL.md is not a regular file",
				SkillFile:  skillFile,
			})
			continue
		}

		name, description, err := readMetadata(skillFile, entry.Name())
		if err != nil {
			return nil, err
		}

		skills = append(skills, diskSkill{
			ID:          entry.Name(),
			Path:        entryPath,
			State:       StateActive,
			Source:      source,
			LinkTarget:  linkTarget,
			Name:        name,
			Description: description,
			SkillFile:   skillFile,
		})
	}

	return skills, nil
}
