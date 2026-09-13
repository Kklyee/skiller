package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	activeDirEnv   = "SKILLER_ACTIVE_DIR"
	disabledDirEnv = "SKILLER_DISABLED_DIR"
)

type Set struct {
	Active   string
	Disabled string
}

func Default() (Set, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Set{}, fmt.Errorf("resolve user home directory: %w", err)
	}

	active := filepath.Join(home, ".agents", "skills")
	disabled := filepath.Join(home, ".skiller", "disabled")

	if value, ok := os.LookupEnv(activeDirEnv); ok && value != "" {
		active = value
	}

	if value, ok := os.LookupEnv(disabledDirEnv); ok && value != "" {
		disabled = value
	}

	return Set{
		Active:   active,
		Disabled: disabled,
	}, nil
}
