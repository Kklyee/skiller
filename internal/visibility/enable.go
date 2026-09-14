package visibility

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Kklyee/skiller/internal/catalog"
)

func Enable(activeDir, disabledDir, id string) (bool, error) {
	return withLock(disabledDir, func() (bool, error) {
		return EnableWithoutLock(activeDir, disabledDir, id)
	})
}

func EnableWithoutLock(activeDir, disabledDir, id string) (bool, error) {
	skills, err := catalog.Scan(activeDir, disabledDir)
	if err != nil {
		return false, fmt.Errorf("inspect installed skills: %w", err)
	}

	skill, found := findSkill(skills, id)
	if !found {
		return false, fmt.Errorf("skill %q is not installed", id)
	}

	switch skill.State {
	case catalog.StateActive:
		return false, nil

	case catalog.StateConflict:
		return false, fmt.Errorf(
			"skill %q is in conflict: it exists in both active and disabled locations",
			id,
		)

	case catalog.StateDisabled:

	default:
		return false, fmt.Errorf(
			"skill %q has unsupported state %s",
			id,
			skill.State,
		)
	}

	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		return false, fmt.Errorf(
			"create active skills directory %q: %w",
			activeDir,
			err,
		)
	}

	destination := filepath.Join(activeDir, skill.ID)

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

	if err := os.Rename(skill.DisabledPath, destination); err != nil {
		return false, fmt.Errorf(
			"enable skill %q: move %q to %q: %w",
			id,
			skill.DisabledPath,
			destination,
			err,
		)
	}

	return true, nil
}
