package environment

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

type Kind string

const (
	KindGroup   Kind = "group"
	KindProfile Kind = "profile"
	KindProject Kind = "project"
)

type Target struct {
	Kind Kind   `toml:"kind"`
	Name string `toml:"name"`
	Path string `toml:"path,omitempty"`
}

type Status string

const (
	StatusManual         Status = "manual"
	StatusSynced         Status = "synced"
	StatusModified       Status = "modified"
	StatusNeedsAttention Status = "needs-attention"
)

type Store struct {
	Path string
}

func New(path string) Store {
	return Store{Path: path}
}

func (s Store) Load() (Target, bool, error) {
	if s.Path == "" {
		return Target{}, false, nil
	}

	var target Target
	if _, err := toml.DecodeFile(s.Path, &target); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Target{}, false, nil
		}
		return Target{}, false, fmt.Errorf("read environment state %q: %w", s.Path, err)
	}
	if err := target.Validate(); err != nil {
		return Target{}, false, fmt.Errorf("validate environment state %q: %w", s.Path, err)
	}
	return target, true, nil
}

func (s Store) Save(target Target) error {
	if s.Path == "" {
		return errors.New("environment state path is empty")
	}
	if err := target.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("create environment state directory: %w", err)
	}

	var data bytes.Buffer
	if err := toml.NewEncoder(&data).Encode(target); err != nil {
		return fmt.Errorf("encode environment state: %w", err)
	}
	if err := os.WriteFile(s.Path, data.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write environment state %q: %w", s.Path, err)
	}
	return nil
}

func (s Store) Clear() error {
	if s.Path == "" {
		return nil
	}
	if err := os.Remove(s.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove environment state %q: %w", s.Path, err)
	}
	return nil
}

func (t Target) Validate() error {
	switch t.Kind {
	case KindGroup, KindProfile:
		if t.Path != "" {
			return fmt.Errorf("%s target must not have a path", t.Kind)
		}
	case KindProject:
		if strings.TrimSpace(t.Path) == "" {
			return errors.New("project target path must not be empty")
		}
	default:
		return fmt.Errorf("unsupported environment target kind %q", t.Kind)
	}
	if err := validateName(t.Name); err != nil {
		return err
	}
	return nil
}

func Evaluate(loaded, hasIssues bool, changes int) Status {
	if !loaded {
		return StatusManual
	}
	if hasIssues {
		return StatusNeedsAttention
	}
	if changes > 0 {
		return StatusModified
	}
	return StatusSynced
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("environment target name must not be empty")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("environment target name %q contains a control character", name)
		}
	}
	return nil
}
