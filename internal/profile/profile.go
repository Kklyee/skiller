package profile

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
)

type Profile struct {
	Name    string   `toml:"name"`
	Groups  []string `toml:"groups"`
	Skills  []string `toml:"skills"`
	Exclude []string `toml:"exclude"`
}

type Target struct {
	Profile       Profile
	Group         group.Group
	MissingGroups []string
}

type Store struct {
	Dir string
}

func New(dir string) Store {
	return Store{Dir: dir}
}

func (s Store) List() ([]Profile, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read profiles directory %q: %w", s.Dir, err)
	}

	profiles := make([]Profile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		stored, err := s.Get(name)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, stored)
	}

	slices.SortFunc(profiles, func(a, b Profile) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return profiles, nil
}

func (s Store) Get(name string) (Profile, error) {
	if err := validateName(name); err != nil {
		return Profile{}, err
	}

	path := s.path(name)
	var stored Profile
	if _, err := toml.DecodeFile(path, &stored); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Profile{}, fmt.Errorf("profile %q does not exist", name)
		}
		return Profile{}, fmt.Errorf("read profile %q: %w", name, err)
	}
	if stored.Name != name {
		return Profile{}, fmt.Errorf("profile file %q has name %q", path, stored.Name)
	}
	if err := validateLists(stored); err != nil {
		return Profile{}, fmt.Errorf("validate profile %q: %w", name, err)
	}

	return stored, nil
}

func (s Store) Create(name string) (Profile, error) {
	if err := validateName(name); err != nil {
		return Profile{}, err
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return Profile{}, fmt.Errorf("create profiles directory %q: %w", s.Dir, err)
	}

	path := s.path(name)
	if _, err := os.Lstat(path); err == nil {
		return Profile{}, fmt.Errorf("profile %q already exists", name)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Profile{}, fmt.Errorf("inspect profile %q: %w", name, err)
	}

	stored := Profile{Name: name, Groups: []string{}, Skills: []string{}, Exclude: []string{}}
	if err := writeProfile(path, stored, true); err != nil {
		return Profile{}, fmt.Errorf("create profile %q: %w", name, err)
	}
	return stored, nil
}

func (s Store) Save(stored Profile, replace bool) error {
	if err := validateName(stored.Name); err != nil {
		return err
	}
	if err := validateLists(stored); err != nil {
		return fmt.Errorf("validate profile %q: %w", stored.Name, err)
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("create profiles directory %q: %w", s.Dir, err)
	}

	path := s.path(stored.Name)
	if !replace {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("profile %q already exists", stored.Name)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("inspect profile %q: %w", stored.Name, err)
		}
	}
	if err := writeProfile(path, stored, !replace); err != nil {
		return fmt.Errorf("save profile %q: %w", stored.Name, err)
	}
	return nil
}

func (s Store) Update(name string, updated Profile) error {
	if err := validateName(name); err != nil {
		return err
	}
	if updated.Name != "" && updated.Name != name {
		return fmt.Errorf("profile name cannot change from %q to %q", name, updated.Name)
	}
	updated.Name = name
	if err := validateLists(updated); err != nil {
		return fmt.Errorf("validate profile %q: %w", name, err)
	}
	if _, err := s.Get(name); err != nil {
		return err
	}
	if err := writeProfile(s.path(name), updated, false); err != nil {
		return fmt.Errorf("update profile %q: %w", name, err)
	}
	return nil
}

func (s Store) Delete(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if err := os.Remove(s.path(name)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("profile %q does not exist", name)
		}
		return fmt.Errorf("delete profile %q: %w", name, err)
	}
	return nil
}

func Resolve(stored Profile, groups []group.Group) Target {
	groupMap := make(map[string]group.Group, len(groups))
	for _, current := range groups {
		groupMap[current.Name] = current
	}

	desired := make(map[string]struct{}, len(stored.Skills))
	for _, id := range stored.Skills {
		desired[id] = struct{}{}
	}
	missingGroups := make([]string, 0)
	for _, name := range stored.Groups {
		current, ok := groupMap[name]
		if !ok {
			missingGroups = append(missingGroups, name)
			continue
		}
		for _, id := range current.Skills {
			desired[id] = struct{}{}
		}
	}

	excluded := make(map[string]struct{}, len(stored.Exclude))
	for _, id := range stored.Exclude {
		excluded[id] = struct{}{}
		delete(desired, id)
	}

	ids := make([]string, 0, len(desired))
	for id := range desired {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	return Target{
		Profile:       stored,
		Group:         group.Group{Name: stored.Name, Skills: ids},
		MissingGroups: missingGroups,
	}
}

func MissingSkills(target Target, installed []catalog.Skill) []string {
	return group.MissingSkills(target.Group, installed)
}

func (s Store) path(name string) string {
	return filepath.Join(s.Dir, name+".toml")
}

func writeProfile(path string, profile Profile, exclusive bool) error {
	var data bytes.Buffer
	if err := toml.NewEncoder(&data).Encode(profile); err != nil {
		return fmt.Errorf("encode TOML: %w", err)
	}
	flags := os.O_WRONLY | os.O_CREATE
	if exclusive {
		flags |= os.O_EXCL
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(data.Bytes()); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func validateName(name string) error {
	if name == "" || name == "." || name == ".." {
		return errors.New("profile name must not be empty or a path marker")
	}
	if strings.ContainsAny(name, `/\\`) {
		return fmt.Errorf("profile name %q must not contain path separators", name)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("profile name %q contains a control character", name)
		}
	}
	return nil
}

func validateLists(profile Profile) error {
	if err := validateUnique(profile.Groups, "group"); err != nil {
		return err
	}
	if err := validateUnique(profile.Skills, "skill"); err != nil {
		return err
	}
	return validateUnique(profile.Exclude, "excluded skill")
}

func validateUnique(values []string, kind string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", kind)
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("%s %q is listed more than once", kind, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}
