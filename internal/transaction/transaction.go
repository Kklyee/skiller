package transaction

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Kklyee/skiller/internal/lock"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/Kklyee/skiller/internal/visibility"
)

type journal struct {
	Version    int         `json:"version"`
	Group      string      `json:"group"`
	Operations []operation `json:"operations"`
}

type operation struct {
	Action     string `json:"action"`
	ID         string `json:"id"`
	From       string `json:"from"`
	To         string `json:"to"`
	Completed  bool   `json:"completed"`
	RolledBack bool   `json:"rolled_back"`
}

func Apply(pathSet paths.Set, plan reconcile.Plan) error {
	if plan.HasIssues() {
		return errors.New("cannot apply plan with missing skills or catalog issues")
	}
	if plan.Changes() == 0 {
		return nil
	}

	journalPath := pathSet.Journal
	if journalPath == "" {
		journalPath = paths.JournalPath(pathSet.Disabled)
	}
	lockPath := pathSet.Lock
	if lockPath == "" {
		lockPath = paths.LockPath(pathSet.Disabled)
	}

	handle, err := lock.Acquire(lockPath)
	if err != nil {
		return err
	}

	applyErr := applyLocked(pathSet, plan, journalPath)
	closeErr := handle.Close()
	if applyErr != nil {
		return applyErr
	}
	if closeErr != nil {
		return closeErr
	}

	return nil
}

func EnsureNoJournal(path string) error {
	if path == "" {
		return errors.New("transaction journal path is empty")
	}

	_, err := os.Lstat(path)
	switch {
	case err == nil:
		return fmt.Errorf("unfinished transaction journal exists at %s", path)
	case errors.Is(err, fs.ErrNotExist):
		return nil
	default:
		return fmt.Errorf("inspect transaction journal %q: %w", path, err)
	}
}

func applyLocked(pathSet paths.Set, plan reconcile.Plan, journalPath string) error {
	if err := EnsureNoJournal(journalPath); err != nil {
		return err
	}

	state := journal{
		Version: 1,
		Group:   plan.Group,
	}
	for _, id := range plan.Enable {
		state.Operations = append(state.Operations, operation{
			Action: "enable",
			ID:     id,
			From:   filepath.Join(pathSet.Disabled, id),
			To:     filepath.Join(pathSet.Active, id),
		})
	}
	for _, id := range plan.Disable {
		state.Operations = append(state.Operations, operation{
			Action: "disable",
			ID:     id,
			From:   filepath.Join(pathSet.Active, id),
			To:     filepath.Join(pathSet.Disabled, id),
		})
	}

	if err := writeJournal(journalPath, state); err != nil {
		return fmt.Errorf("create transaction journal: %w", err)
	}

	for index := range state.Operations {
		op := &state.Operations[index]
		var changed bool
		var err error
		switch op.Action {
		case "enable":
			changed, err = visibility.EnableWithoutLock(pathSet.Active, pathSet.Disabled, op.ID)
		case "disable":
			changed, err = visibility.DisableWithoutLock(pathSet.Active, pathSet.Disabled, op.ID)
		default:
			err = fmt.Errorf("unsupported transaction action %q", op.Action)
		}
		if err != nil {
			return rollback(journalPath, state, fmt.Errorf("%s %q: %w", op.Action, op.ID, err))
		}
		if !changed {
			return rollback(journalPath, state, fmt.Errorf("%s %q made no change", op.Action, op.ID))
		}

		op.Completed = true
		if err := writeJournal(journalPath, state); err != nil {
			return rollback(journalPath, state, fmt.Errorf("record completed %s %q: %w", op.Action, op.ID, err))
		}
		if err := verifyMove(op.From, op.To); err != nil {
			return rollback(journalPath, state, fmt.Errorf("verify %s %q: %w", op.Action, op.ID, err))
		}
	}

	if err := os.Remove(journalPath); err != nil {
		return fmt.Errorf("remove completed transaction journal: %w", err)
	}

	return nil
}

func rollback(journalPath string, state journal, cause error) error {
	var rollbackErr error
	for index := len(state.Operations) - 1; index >= 0; index-- {
		op := &state.Operations[index]
		if !op.Completed {
			continue
		}

		if err := reverseMove(op.To, op.From); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("rollback %s %q: %w", op.Action, op.ID, err))
			continue
		}
		op.RolledBack = true
		if err := writeJournal(journalPath, state); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("record rollback %s %q: %w", op.Action, op.ID, err))
		}
	}

	if rollbackErr != nil {
		return fmt.Errorf("%w; rollback failed: %v", cause, rollbackErr)
	}
	if err := os.Remove(journalPath); err != nil {
		return fmt.Errorf("%w; remove rolled-back transaction journal: %v", cause, err)
	}

	return cause
}

func verifyMove(from, to string) error {
	if _, err := os.Lstat(from); !errors.Is(err, fs.ErrNotExist) {
		if err == nil {
			return fmt.Errorf("source still exists: %s", from)
		}
		return fmt.Errorf("inspect source %s: %w", from, err)
	}
	if _, err := os.Lstat(to); err != nil {
		return fmt.Errorf("destination is missing: %s: %w", to, err)
	}

	return nil
}

func reverseMove(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		return fmt.Errorf("refusing to overwrite existing path %s", to)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect rollback destination %s: %w", to, err)
	}
	if _, err := os.Lstat(from); err != nil {
		return fmt.Errorf("inspect rollback source %s: %w", from, err)
	}
	if err := os.Rename(from, to); err != nil {
		return err
	}

	return verifyMove(from, to)
}

func writeJournal(path string, state journal) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	return nil
}
