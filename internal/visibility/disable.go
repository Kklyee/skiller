package visibility

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Kklyee/skiller/internal/catalog"
)

func Disable(activeDir, disabledDir, id string) (bool, error) {
	skills, err := catalog.Scan(activeDir, disabledDir)
	if err != nil {
		return false, fmt.Errorf("inspect installed skills: %w", err)
	}

	skill, found := findSkill(skills, id)
	if !found {
		return false, fmt.Errorf("skill %q is not installed", id)
	}

	switch skill.State {
	case catalog.StateDisabled:
		return false, nil

	case catalog.StateConflict:
		return false, fmt.Errorf(
			"skill %q is in conflict: it exists in both active and disabled locations",
			id,
		)

	case catalog.StateActive:
		// Continue below.

	default:
		return false, fmt.Errorf(
			"skill %q has unsupported state %s",
			id,
			skill.State,
		)
	}

	if err := os.MkdirAll(disabledDir, 0o755); err != nil {
		return false, fmt.Errorf(
			"create disabled skills directory %q: %w",
			disabledDir,
			err,
		)
	}

	destination := filepath.Join(disabledDir, skill.ID)

	if _, err := os.Lstat(destination); err == nil {
		return false, fmt.Errorf(
			"refusing to overwrite existing path %q",
			destination,
		)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf(
			"inspect destination %q: %w",
			destination,
			err,
		)
	}

	if err := os.Rename(skill.ActivePath, destination); err != nil {
		return false, fmt.Errorf(
			"disable skill %q: move %q to %q: %w",
			id,
			skill.ActivePath,
			destination,
			err,
		)
	}

	return true, nil
}

func findSkill(skills []catalog.Skill, id string) (catalog.Skill, bool) {
	for _, skill := range skills {
		if skill.ID == id {
			return skill, true
		}
	}

	return catalog.Skill{}, false
}
