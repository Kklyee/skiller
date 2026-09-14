package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	activeDirEnv   = "SKILLER_ACTIVE_DIR"
	disabledDirEnv = "SKILLER_DISABLED_DIR"
	journalEnv     = "SKILLER_TRANSACTION_JOURNAL"
)

type Set struct {
	Active   string
	Disabled string
	Journal  string
}

func Default() (Set, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Set{}, fmt.Errorf("resolve user home directory: %w", err)
	}

	active := filepath.Join(home, ".agents", "skills")
	disabled := filepath.Join(home, ".skiller", "disabled")
	journal := filepath.Join(home, ".skiller", "transaction.json")

	if value, ok := os.LookupEnv(activeDirEnv); ok && value != "" {
		active = value
	}

	if value, ok := os.LookupEnv(disabledDirEnv); ok && value != "" {
		disabled = value
	}

	if value, ok := os.LookupEnv(journalEnv); ok && value != "" {
		journal = value
	}

	return Set{
		Active:   active,
		Disabled: disabled,
		Journal:  journal,
	}, nil
}
