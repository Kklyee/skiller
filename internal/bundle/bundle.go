package bundle

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
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/profile"
	"github.com/Kklyee/skiller/internal/provenance"
)

const Version = 1

type Bundle struct {
	Version    int                         `toml:"version"`
	Active     []string                    `toml:"active"`
	Pins       []string                    `toml:"pins"`
	Groups     []group.Group               `toml:"groups"`
	Profiles   []profile.Profile           `toml:"profiles"`
	Provenance map[string]provenance.Entry `toml:"provenance"`
}

func (b Bundle) Validate() error {
	if b.Version != Version {
		return fmt.Errorf("unsupported bundle version %d", b.Version)
	}
	if err := validateIDs(b.Active, "active skill"); err != nil {
		return err
	}
	if err := validateIDs(b.Pins, "pinned skill"); err != nil {
		return err
	}
	groupNames := make(map[string]struct{}, len(b.Groups))
	for _, stored := range b.Groups {
		if err := validateName(stored.Name, "group"); err != nil {
			return err
		}
		if _, ok := groupNames[stored.Name]; ok {
			return fmt.Errorf("group %q is listed more than once", stored.Name)
		}
		groupNames[stored.Name] = struct{}{}
		if err := validateIDs(stored.Skills, "group skill"); err != nil {
			return fmt.Errorf("validate group %q: %w", stored.Name, err)
		}
	}
	profileNames := make(map[string]struct{}, len(b.Profiles))
	for _, stored := range b.Profiles {
		if err := validateName(stored.Name, "profile"); err != nil {
			return err
		}
		if _, ok := profileNames[stored.Name]; ok {
			return fmt.Errorf("profile %q is listed more than once", stored.Name)
		}
		profileNames[stored.Name] = struct{}{}
		if err := validateIDs(stored.Groups, "profile group"); err != nil {
			return fmt.Errorf("validate profile %q: %w", stored.Name, err)
		}
		if err := validateIDs(stored.Skills, "profile skill"); err != nil {
			return fmt.Errorf("validate profile %q: %w", stored.Name, err)
		}
		if err := validateIDs(stored.Exclude, "profile excluded skill"); err != nil {
			return fmt.Errorf("validate profile %q: %w", stored.Name, err)
		}
	}
	for id := range b.Provenance {
		if err := validateID(id, "provenance skill"); err != nil {
			return err
		}
	}
	return nil
}

func (b Bundle) Missing(installed []catalog.Skill) []string {
	known := make(map[string]struct{}, len(installed))
	for _, skill := range installed {
		known[skill.ID] = struct{}{}
	}
	wanted := make(map[string]struct{})
	add := func(ids []string) {
		for _, id := range ids {
			if _, ok := known[id]; !ok {
				wanted[id] = struct{}{}
			}
		}
	}
	add(b.Active)
	add(b.Pins)
	for _, stored := range b.Groups {
		add(stored.Skills)
	}
	for _, stored := range b.Profiles {
		add(stored.Skills)
	}
	for id := range b.Provenance {
		if _, ok := known[id]; !ok {
			wanted[id] = struct{}{}
		}
	}

	missing := make([]string, 0, len(wanted))
	for id := range wanted {
		missing = append(missing, id)
	}
	slices.Sort(missing)
	return missing
}

func Write(path string, stored Bundle) error {
	if err := stored.Validate(); err != nil {
		return fmt.Errorf("validate bundle: %w", err)
	}
	var data bytes.Buffer
	if err := toml.NewEncoder(&data).Encode(stored); err != nil {
		return fmt.Errorf("encode bundle: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create bundle directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create bundle %q: %w", path, err)
	}
	if _, err := file.Write(data.Bytes()); err != nil {
		_ = file.Close()
		return fmt.Errorf("write bundle %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close bundle %q: %w", path, err)
	}
	return nil
}

func Load(path string) (Bundle, error) {
	var stored Bundle
	metadata, err := toml.DecodeFile(path, &stored)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Bundle{}, fmt.Errorf("bundle %q does not exist", path)
		}
		return Bundle{}, fmt.Errorf("read bundle %q: %w", path, err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		return Bundle{}, fmt.Errorf("bundle %q has unsupported field %q", path, undecoded[0])
	}
	if err := stored.Validate(); err != nil {
		return Bundle{}, fmt.Errorf("validate bundle %q: %w", path, err)
	}
	return stored, nil
}

func validateIDs(ids []string, kind string) error {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if err := validateID(id, kind); err != nil {
			return err
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%s %q is listed more than once", kind, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateID(id, kind string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s must not be empty", kind)
	}
	return nil
}

func validateName(name, kind string) error {
	if strings.TrimSpace(name) == "" || name == "." || name == ".." {
		return fmt.Errorf("%s name must not be empty or a path marker", kind)
	}
	if strings.ContainsAny(name, `/\\`) {
		return fmt.Errorf("%s name %q must not contain path separators", kind, name)
	}
	return nil
}
