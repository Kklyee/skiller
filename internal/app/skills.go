package app

import (
	"fmt"

	"github.com/Kklyee/skiller/internal/discovery"
	"github.com/Kklyee/skiller/internal/domain"
	"github.com/Kklyee/skiller/internal/storage"
)

type SkillService struct {
	paths   storage.Paths
	scanner *discovery.Scanner
}

func NewSkillService(
	paths storage.Paths,
	scanner *discovery.Scanner,
) *SkillService {
	return &SkillService{
		paths:   paths,
		scanner: scanner,
	}
}

func (s *SkillService) List() ([]domain.Skill, error) {
	skills, err := s.scanner.Scan(
		s.paths.ActiveDir,
		s.paths.DisabledDir,
	)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}

	return skills, nil
}
