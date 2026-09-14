package group

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
)

type Group struct {
	Name    string   `toml:"name"`
	Skills  []string `toml:"skills"`
	Missing []string `toml:"-"`
}

type Store struct {
	Dir string
}

func New(dir string) Store {
	return Store{Dir: dir}
}

func (s Store) List() ([]Group, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read groups directory %q: %w", s.Dir, err)
	}

	groups := make([]Group, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		group, err := s.Get(name)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	slices.SortFunc(groups, func(a, b Group) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return groups, nil
}

func (s Store) ListWithMissing(installed []catalog.Skill) ([]Group, error) {
	groups, err := s.List()
	if err != nil {
		return nil, err
	}

	for index := range groups {
		groups[index].Missing = MissingSkills(groups[index], installed)
	}

	return groups, nil
}

func (s Store) Get(name string) (Group, error) {
	if err := validateName(name); err != nil {
		return Group{}, err
	}

	path := s.path(name)
	var group Group
	if _, err := toml.DecodeFile(path, &group); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Group{}, fmt.Errorf("group %q does not exist", name)
		}
		return Group{}, fmt.Errorf("read group %q: %w", name, err)
	}

	if group.Name != name {
		return Group{}, fmt.Errorf("group file %q has name %q", path, group.Name)
	}
	if err := validateSkills(group.Skills); err != nil {
		return Group{}, fmt.Errorf("validate group %q: %w", name, err)
	}

	return group, nil
}

func (s Store) Create(name string) (Group, error) {
	if err := validateName(name); err != nil {
		return Group{}, err
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return Group{}, fmt.Errorf("create groups directory %q: %w", s.Dir, err)
	}

	path := s.path(name)
	if _, err := os.Lstat(path); err == nil {
		return Group{}, fmt.Errorf("group %q already exists", name)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Group{}, fmt.Errorf("inspect group %q: %w", name, err)
	}

	group := Group{Name: name, Skills: []string{}}
	if err := writeGroup(path, group, true); err != nil {
		return Group{}, fmt.Errorf("create group %q: %w", name, err)
	}

	return group, nil
}

func (s Store) Delete(name string) error {
	if err := validateName(name); err != nil {
		return err
	}

	if err := os.Remove(s.path(name)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("group %q does not exist", name)
		}
		return fmt.Errorf("delete group %q: %w", name, err)
	}

	return nil
}

func (s Store) Add(name string, skills ...string) (int, error) {
	group, err := s.Get(name)
	if err != nil {
		return 0, err
	}
	if err := validateSkillIDs(skills); err != nil {
		return 0, err
	}

	known := make(map[string]struct{}, len(group.Skills))
	for _, skill := range group.Skills {
		known[skill] = struct{}{}
	}

	added := 0
	for _, skill := range skills {
		if _, ok := known[skill]; ok {
			continue
		}
		known[skill] = struct{}{}
		group.Skills = append(group.Skills, skill)
		added++
	}

	if added == 0 {
		return 0, nil
	}
	if err := writeGroup(s.path(name), group, false); err != nil {
		return 0, fmt.Errorf("update group %q: %w", name, err)
	}

	return added, nil
}

func (s Store) Remove(name string, skills ...string) (int, error) {
	group, err := s.Get(name)
	if err != nil {
		return 0, err
	}
	if err := validateSkillIDs(skills); err != nil {
		return 0, err
	}

	remove := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		remove[skill] = struct{}{}
	}

	kept := make([]string, 0, len(group.Skills))
	removed := 0
	for _, skill := range group.Skills {
		if _, ok := remove[skill]; ok {
			removed++
			continue
		}
		kept = append(kept, skill)
	}

	if removed == 0 {
		return 0, nil
	}
	group.Skills = kept
	if err := writeGroup(s.path(name), group, false); err != nil {
		return 0, fmt.Errorf("update group %q: %w", name, err)
	}

	return removed, nil
}

func MissingSkills(group Group, installed []catalog.Skill) []string {
	known := make(map[string]struct{}, len(installed))
	for _, skill := range installed {
		known[skill.ID] = struct{}{}
	}

	missing := make([]string, 0)
	for _, skill := range group.Skills {
		if _, ok := known[skill]; !ok {
			missing = append(missing, skill)
		}
	}

	return missing
}

func (s Store) path(name string) string {
	return filepath.Join(s.Dir, name+".toml")
}

func writeGroup(path string, group Group, exclusive bool) error {
	var data bytes.Buffer
	if err := toml.NewEncoder(&data).Encode(group); err != nil {
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
	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

func validateName(name string) error {
	if name == "" || name == "." || name == ".." {
		return errors.New("group name must not be empty or a path marker")
	}
	if strings.ContainsAny(name, `/\\`) {
		return fmt.Errorf("group name %q must not contain path separators", name)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("group name %q contains a control character", name)
		}
	}

	return nil
}

func validateSkills(skills []string) error {
	seen := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		if err := validateSkillIDs([]string{skill}); err != nil {
			return err
		}
		if _, ok := seen[skill]; ok {
			return fmt.Errorf("skill %q is listed more than once", skill)
		}
		seen[skill] = struct{}{}
	}

	return nil
}

func validateSkillIDs(skills []string) error {
	for _, skill := range skills {
		if strings.TrimSpace(skill) == "" {
			return errors.New("skill ID must not be empty")
		}
	}

	return nil
}
