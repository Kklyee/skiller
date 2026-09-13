package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

type Paths struct {
	ActiveDir   string
	DisabledDir string
}

func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("get user home directory: %w", err)
	}

	activeDir := filepath.Join(home, ".agents", "skills")
	disabledDir := filepath.Join(home, ".skiller", "disabled")

	if value := os.Getenv("SKILLER_ACTIVE_DIR"); value != "" {
	 activeDir = value
	}
	if value := os.Getenv("SKILLER_DISABLED_DIR"); value != "" {
	 disabledDir = value
	}

	return Paths{
		ActiveDir:   activeDir,
		DisabledDir: disabledDir,
	}, nil
}
