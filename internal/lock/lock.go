package lock

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Handle struct {
	path string
	file *os.File
}

func Acquire(path string) (*Handle, error) {
	if path == "" {
		return nil, errors.New("lock path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("another Skiller process is modifying skills")
		}
		return nil, fmt.Errorf("acquire lock %q: %w", path, err)
	}

	if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("write lock %q: %w", path, err)
	}

	return &Handle{path: path, file: file}, nil
}

func (h *Handle) Close() error {
	if h == nil || h.file == nil {
		return nil
	}

	if err := h.file.Close(); err != nil {
		return fmt.Errorf("close lock %q: %w", h.path, err)
	}
	if err := os.Remove(h.path); err != nil {
		return fmt.Errorf("release lock %q: %w", h.path, err)
	}

	h.file = nil
	return nil
}
