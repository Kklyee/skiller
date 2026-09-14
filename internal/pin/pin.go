package pin

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

type Store struct {
	Path string
}

type file struct {
	Skills []string `toml:"skills"`
}

func New(path string) Store {
	return Store{Path: path}
}

func (s Store) List() ([]string, error) {
	if s.Path == "" {
		return []string{}, nil
	}

	var stored file
	if _, err := toml.DecodeFile(s.Path, &stored); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read pins file %q: %w", s.Path, err)
	}
	if err := validateStored(stored.Skills); err != nil {
		return nil, fmt.Errorf("validate pins file %q: %w", s.Path, err)
	}

	pins := slices.Clone(stored.Skills)
	slices.Sort(pins)
	return pins, nil
}

func (s Store) Add(skills ...string) (int, error) {
	if err := validate(skills); err != nil {
		return 0, err
	}
	if s.Path == "" {
		return 0, errors.New("pins file path is empty")
	}

	pins, err := s.List()
	if err != nil {
		return 0, err
	}
	known := make(map[string]struct{}, len(pins))
	for _, id := range pins {
		known[id] = struct{}{}
	}

	added := 0
	for _, id := range skills {
		if _, ok := known[id]; ok {
			continue
		}
		known[id] = struct{}{}
		pins = append(pins, id)
		added++
	}
	if added == 0 {
		return 0, nil
	}

	slices.Sort(pins)
	if err := s.write(file{Skills: pins}); err != nil {
		return 0, err
	}
	return added, nil
}

func (s Store) Remove(skills ...string) (int, error) {
	if err := validate(skills); err != nil {
		return 0, err
	}
	if s.Path == "" {
		return 0, errors.New("pins file path is empty")
	}

	pins, err := s.List()
	if err != nil {
		return 0, err
	}
	remove := make(map[string]struct{}, len(skills))
	for _, id := range skills {
		remove[id] = struct{}{}
	}

	kept := make([]string, 0, len(pins))
	removed := 0
	for _, id := range pins {
		if _, ok := remove[id]; ok {
			removed++
			continue
		}
		kept = append(kept, id)
	}
	if removed == 0 {
		return 0, nil
	}

	if err := s.write(file{Skills: kept}); err != nil {
		return 0, err
	}
	return removed, nil
}

func (s Store) write(stored file) error {
	var data bytes.Buffer
	if err := toml.NewEncoder(&data).Encode(stored); err != nil {
		return fmt.Errorf("encode pins: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("create pins directory: %w", err)
	}
	if err := os.WriteFile(s.Path, data.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write pins file %q: %w", s.Path, err)
	}
	return nil
}

func validate(skills []string) error {
	for _, id := range skills {
		if strings.TrimSpace(id) == "" {
			return errors.New("pinned skill ID must not be empty")
		}
	}
	return nil
}

func validateStored(skills []string) error {
	if err := validate(skills); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(skills))
	for _, id := range skills {
		if _, ok := seen[id]; ok {
			return fmt.Errorf("skill %q is pinned more than once", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}
