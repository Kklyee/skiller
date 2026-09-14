//go:build !windows

package doctor

import (
	"fmt"
	"os"
	"syscall"
)

func sameVolume(first, second string) (bool, error) {
	firstInfo, err := os.Stat(first)
	if err != nil {
		return false, fmt.Errorf("stat %q: %w", first, err)
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		return false, fmt.Errorf("stat %q: %w", second, err)
	}

	firstStat, ok := firstInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("unsupported filesystem metadata for %q", first)
	}
	secondStat, ok := secondInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("unsupported filesystem metadata for %q", second)
	}

	return firstStat.Dev == secondStat.Dev, nil
}
