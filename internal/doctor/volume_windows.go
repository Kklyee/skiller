package doctor

import (
	"fmt"
	"path/filepath"
	"strings"
)

func sameVolume(first, second string) (bool, error) {
	firstVolume := filepath.VolumeName(first)
	secondVolume := filepath.VolumeName(second)
	if firstVolume == "" || secondVolume == "" {
		return false, fmt.Errorf("unable to determine volume for %q and %q", first, second)
	}

	return strings.EqualFold(firstVolume, secondVolume), nil
}
