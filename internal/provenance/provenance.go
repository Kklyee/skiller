package provenance

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

type Entry struct {
	Source     string `toml:"source"`
	Repository string `toml:"repository"`
	Installer  string `toml:"installer"`
	Revision   string `toml:"revision"`
}

type Store struct {
	Path string
}

type file struct {
	Skills map[string]Entry `toml:"skills"`
}

func New(path string) Store {
	return Store{Path: path}
}

func (s Store) List() (map[string]Entry, error) {
	if s.Path == "" {
		return map[string]Entry{}, nil
	}

	var stored file
	if _, err := toml.DecodeFile(s.Path, &stored); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]Entry{}, nil
		}
		return nil, fmt.Errorf("read provenance file %q: %w", s.Path, err)
	}
	if stored.Skills == nil {
		stored.Skills = map[string]Entry{}
	}
	for id := range stored.Skills {
		if err := validateID(id); err != nil {
			return nil, fmt.Errorf("validate provenance file %q: %w", s.Path, err)
		}
	}
	return stored.Skills, nil
}

func (s Store) Get(id string) (Entry, bool, error) {
	if err := validateID(id); err != nil {
		return Entry{}, false, err
	}
	entries, err := s.List()
	if err != nil {
		return Entry{}, false, err
	}
	entry, ok := entries[id]
	return entry, ok, nil
}

func (s Store) Set(id string, entry Entry) error {
	if err := validateID(id); err != nil {
		return err
	}
	if s.Path == "" {
		return errors.New("provenance file path is empty")
	}
	entries, err := s.List()
	if err != nil {
		return err
	}
	entries[id] = entry
	return s.write(file{Skills: entries})
}

func (s Store) Remove(id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	if s.Path == "" {
		return errors.New("provenance file path is empty")
	}
	entries, err := s.List()
	if err != nil {
		return err
	}
	if _, ok := entries[id]; !ok {
		return nil
	}
	delete(entries, id)
	return s.write(file{Skills: entries})
}

func (s Store) write(stored file) error {
	var data bytes.Buffer
	if err := toml.NewEncoder(&data).Encode(stored); err != nil {
		return fmt.Errorf("encode provenance: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("create provenance directory: %w", err)
	}
	if err := os.WriteFile(s.Path, data.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write provenance file %q: %w", s.Path, err)
	}
	return nil
}

func validateID(id string) error {
	if id == "" || id == "." || id == ".." {
		return errors.New("skill ID must not be empty or a path marker")
	}
	if strings.ContainsAny(id, `/\\`) {
		return fmt.Errorf("skill ID %q must not contain path separators", id)
	}
	for _, r := range id {
		if unicode.IsControl(r) {
			return fmt.Errorf("skill ID %q contains a control character", id)
		}
	}
	return nil
}
