package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	activeDirEnv   = "SKILLER_ACTIVE_DIR"
	disabledDirEnv = "SKILLER_DISABLED_DIR"
	groupsDirEnv   = "SKILLER_GROUPS_DIR"
	profilesDirEnv = "SKILLER_PROFILES_DIR"
	journalEnv     = "SKILLER_TRANSACTION_JOURNAL"
	lockEnv        = "SKILLER_LOCK"
)

type Set struct {
	Active   string
	Disabled string
	Groups   string
	Profiles string
	Journal  string
	Lock     string
}

func Default() (Set, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Set{}, fmt.Errorf("resolve user home directory: %w", err)
	}

	active := filepath.Join(home, ".agents", "skills")
	disabled := filepath.Join(home, ".skiller", "disabled")
	groups := filepath.Join(home, ".skiller", "groups")
	profiles := filepath.Join(home, ".skiller", "profiles")

	if value, ok := os.LookupEnv(activeDirEnv); ok && value != "" {
		active = value
	}

	if value, ok := os.LookupEnv(disabledDirEnv); ok && value != "" {
		disabled = value
	}

	if value, ok := os.LookupEnv(groupsDirEnv); ok && value != "" {
		groups = value
	}

	if value, ok := os.LookupEnv(profilesDirEnv); ok && value != "" {
		profiles = value
	}

	return Set{
		Active:   active,
		Disabled: disabled,
		Groups:   groups,
		Profiles: profiles,
		Journal:  JournalPath(disabled),
		Lock:     LockPath(disabled),
	}, nil
}

func JournalPath(disabledDir string) string {
	if value, ok := os.LookupEnv(journalEnv); ok && value != "" {
		return value
	}

	return filepath.Join(filepath.Dir(disabledDir), "transaction.json")
}

func LockPath(disabledDir string) string {
	if value, ok := os.LookupEnv(lockEnv); ok && value != "" {
		return value
	}

	return filepath.Join(filepath.Dir(disabledDir), "lock")
}
