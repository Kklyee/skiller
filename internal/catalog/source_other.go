//go:build !windows

package catalog

import "os"

func linkSource(path string) (Source, string, bool) {
	target, err := os.Readlink(path)
	if err != nil {
		return SourceDirectory, "", false
	}

	return SourceSymlink, target, true
}
