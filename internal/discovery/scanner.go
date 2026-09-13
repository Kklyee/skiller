package discovery

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Kklyee/skiller/internal/domain"
)

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Scan(activeDir, disabledDir string) ([]domain.Skill, error) {
	var skills []domain.Skill
	activeSkills, err := s.ScanDir(activeDir, domain.SkillStateActive)

	if err != nil {
		return nil, fmt.Errorf("scan active skills: %w", err)
	}

	disabledSkills, err := s.ScanDir(disabledDir, domain.SkillStateDisabled)
	if err != nil {
		return nil, fmt.Errorf("scan disabled skills: %w", err)
	}

	skills = append(skills, activeSkills...)
	skills = append(skills, disabledSkills...)

	slices.SortFunc(skills, func(a, b domain.Skill) int { return cmp.Compare(a.ID, b.ID) })

	return skills, nil
}

func (s *Scanner) ScanDir(dir string, skillState domain.SkillState) ([]domain.Skill, error) {
	entries, err := os.ReadDir(dir)

	if os.IsNotExist(err) {
		return []domain.Skill{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	var skills []domain.Skill

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()
		skillDir := filepath.Join(dir, skillName)
		skillFile := filepath.Join(skillDir, "SKILL.md")

		if _, err := os.Stat(skillFile); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("stat skill file %s: %w", skillFile, err)
		}

		skill := domain.Skill{
			ID:    skillName,
			State: skillState,
			Path:  skillDir,
		}
		skills = append(skills, skill)

	}
	return skills, nil
}
