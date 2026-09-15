package removal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/lock"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/pin"
	"github.com/Kklyee/skiller/internal/profile"
	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
	"github.com/Kklyee/skiller/internal/transaction"
)

type Result struct {
	Deleted           []string
	PinsRemoved       int
	GroupsUpdated     int
	ProfilesUpdated   int
	ProvenanceRemoved int
}

func Delete(pathSet paths.Set, ids ...string) (Result, error) {
	ids, err := normalizeIDs(ids)
	if err != nil {
		return Result{}, err
	}
	if len(ids) == 0 {
		return Result{}, errors.New("select at least one skill to delete")
	}

	lockPath := pathSet.Lock
	if lockPath == "" {
		lockPath = paths.LockPath(pathSet.Disabled)
	}
	handle, err := lock.Acquire(lockPath)
	if err != nil {
		return Result{}, err
	}

	result, operationErr := deleteLocked(pathSet, ids)
	closeErr := handle.Close()
	if operationErr != nil {
		if closeErr != nil {
			return result, errors.Join(operationErr, closeErr)
		}
		return result, operationErr
	}
	if closeErr != nil {
		return result, closeErr
	}
	return result, nil
}

func deleteLocked(pathSet paths.Set, ids []string) (Result, error) {
	journalPath := pathSet.Journal
	if journalPath == "" {
		journalPath = paths.JournalPath(pathSet.Disabled)
	}
	if err := transaction.EnsureNoJournal(journalPath); err != nil {
		return Result{}, err
	}

	skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
	if err != nil {
		return Result{}, fmt.Errorf("inspect installed skills: %w", err)
	}
	byID := make(map[string]catalog.Skill, len(skills))
	for _, skill := range skills {
		byID[skill.ID] = skill
	}

	groups, err := group.New(pathSet.Groups).List()
	if err != nil {
		return Result{}, err
	}
	profiles, err := profile.New(pathSet.Profiles).List()
	if err != nil {
		return Result{}, err
	}
	pinStore := pin.New(pathSet.Pins)
	pinned, err := pinStore.List()
	if err != nil {
		return Result{}, err
	}
	provenanceStore := skillprovenance.New(pathSet.Provenance)
	provenance, err := provenanceStore.List()
	if err != nil {
		return Result{}, err
	}

	seenPaths := make(map[string]struct{}, len(ids)*2)
	for _, id := range ids {
		skill, ok := byID[id]
		if !ok {
			return Result{}, fmt.Errorf("skill %q is not installed", id)
		}
		for _, target := range []struct {
			root string
			path string
		}{
			{root: pathSet.Active, path: skill.ActivePath},
			{root: pathSet.Disabled, path: skill.DisabledPath},
		} {
			if target.path == "" {
				continue
			}
			expected := filepath.Join(target.root, id)
			if filepath.Clean(target.path) != filepath.Clean(expected) {
				return Result{}, fmt.Errorf("refusing to delete skill %q outside its skill directory", id)
			}
			if _, err := os.Lstat(target.path); err != nil {
				return Result{}, fmt.Errorf("inspect skill %q: %w", id, err)
			}
			if _, ok := seenPaths[target.path]; ok {
				continue
			}
			seenPaths[target.path] = struct{}{}
		}
	}

	result := Result{}
	for _, id := range ids {
		skill := byID[id]
		for _, path := range []string{skill.ActivePath, skill.DisabledPath} {
			if path == "" {
				continue
			}
			if _, ok := seenPaths[path]; !ok {
				continue
			}
			if err := os.RemoveAll(path); err != nil {
				return result, fmt.Errorf("delete skill %q: %w", id, err)
			}
			if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
				if err == nil {
					return result, fmt.Errorf("delete skill %q: path still exists", id)
				}
				return result, fmt.Errorf("verify deleted skill %q: %w", id, err)
			}
			delete(seenPaths, path)
		}
		result.Deleted = append(result.Deleted, id)
	}

	removeSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		removeSet[id] = struct{}{}
	}
	groupStore := group.New(pathSet.Groups)
	for _, stored := range groups {
		if !containsAny(stored.Skills, removeSet) {
			continue
		}
		if _, err := groupStore.Remove(stored.Name, ids...); err != nil {
			return result, fmt.Errorf("clean skill references from group %q: %w", stored.Name, err)
		}
		result.GroupsUpdated++
	}

	profileStore := profile.New(pathSet.Profiles)
	for _, stored := range profiles {
		updated := stored
		var skillsChanged, excludeChanged bool
		updated.Skills, skillsChanged = without(updated.Skills, removeSet)
		updated.Exclude, excludeChanged = without(updated.Exclude, removeSet)
		if !skillsChanged && !excludeChanged {
			continue
		}
		if err := profileStore.Update(stored.Name, updated); err != nil {
			return result, fmt.Errorf("clean skill references from profile %q: %w", stored.Name, err)
		}
		result.ProfilesUpdated++
	}

	pinnedSet := make(map[string]struct{}, len(pinned))
	for _, id := range pinned {
		pinnedSet[id] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := pinnedSet[id]; ok {
			result.PinsRemoved++
		}
	}
	if result.PinsRemoved > 0 {
		if _, err := pinStore.Remove(ids...); err != nil {
			return result, fmt.Errorf("clean deleted skills from pins: %w", err)
		}
	}

	for _, id := range ids {
		if _, ok := provenance[id]; !ok {
			continue
		}
		if err := provenanceStore.Remove(id); err != nil {
			return result, fmt.Errorf("clean provenance for %q: %w", id, err)
		}
		result.ProvenanceRemoved++
	}

	return result, nil
}

func normalizeIDs(ids []string) ([]string, error) {
	unique := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if err := validateID(id); err != nil {
			return nil, err
		}
		unique[id] = struct{}{}
	}
	normalized := make([]string, 0, len(unique))
	for id := range unique {
		normalized = append(normalized, id)
	}
	slices.Sort(normalized)
	return normalized, nil
}

func validateID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." {
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

func containsAny(values []string, wanted map[string]struct{}) bool {
	for _, value := range values {
		if _, ok := wanted[value]; ok {
			return true
		}
	}
	return false
}

func without(values []string, remove map[string]struct{}) ([]string, bool) {
	kept := make([]string, 0, len(values))
	changed := false
	for _, value := range values {
		if _, ok := remove[value]; ok {
			changed = true
			continue
		}
		kept = append(kept, value)
	}
	if !changed {
		return values, false
	}
	return kept, true
}
