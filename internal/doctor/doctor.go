package doctor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/pin"
)

type Level uint8

const (
	Healthy Level = iota + 1
	Warning
	Error
)

func (l Level) String() string {
	switch l {
	case Healthy:
		return "healthy"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

func (l Level) Symbol() string {
	switch l {
	case Healthy:
		return "✓"
	case Warning:
		return "!"
	case Error:
		return "×"
	default:
		return "?"
	}
}

type Check struct {
	Section string
	Name    string
	Level   Level
	Detail  string
}

type Report struct {
	Checks  []Check
	Overall Level
	Summary catalog.Summary
}

func Inspect(pathSet paths.Set) Report {
	report := Report{Overall: Healthy}

	active := inspectPath("Active directory", pathSet.Active)
	disabled := inspectPath("Disabled directory", pathSet.Disabled)
	report.add("Paths", active.check)
	report.add("Paths", disabled.check)

	writableLevel := Healthy
	writableDetail := "both skill directories are writable"
	if !active.writable || !disabled.writable {
		writableLevel = Error
		writableDetail = "one or more skill directories are not writable"
	} else if !active.exists || !disabled.exists {
		writableLevel = Warning
		writableDetail = "missing skill directories can be created"
	}
	report.add("Filesystem", Check{
		Name:   "Writable",
		Level:  writableLevel,
		Detail: writableDetail,
	})

	volumeLevel, volumeDetail := compareVolumes(active, disabled)
	report.add("Filesystem", Check{
		Name:   "Same volume",
		Level:  volumeLevel,
		Detail: volumeDetail,
	})

	renameLevel, renameDetail := checkRename(active)
	report.add("Filesystem", Check{
		Name:   "Rename supported",
		Level:  renameLevel,
		Detail: renameDetail,
	})

	skills, err := catalog.Scan(pathSet.Active, pathSet.Disabled)
	if err != nil {
		report.add("Skills", Check{
			Name:   "Scan",
			Level:  Error,
			Detail: err.Error(),
		})
	} else {
		report.Summary = catalog.Summarize(skills)
		report.Summary.ActiveDir = pathSet.Active
		report.Summary.DisabledDir = pathSet.Disabled
		report.add("Skills", skillCheck("Active", report.Summary.Active))
		report.add("Skills", skillCheck("Disabled", report.Summary.Disabled))
		report.add("Skills", issueCheck("Conflicts", report.Summary.Conflict))
		report.add("Skills", issueCheck("Broken links", report.Summary.Broken))
		report.add("Skills", issueCheck("Invalid directories", report.Summary.Invalid))
		report.add("Skills", pinsCheck(pathSet.Pins, skills))
	}

	report.add("Transactions", journalCheck(pathSet.Journal))

	return report
}

func pinsCheck(path string, skills []catalog.Skill) Check {
	pinned, err := pin.New(path).List()
	if err != nil {
		return Check{Name: "Pinned skills", Level: Error, Detail: err.Error()}
	}

	byID := make(map[string]catalog.Skill, len(skills))
	for _, skill := range skills {
		byID[skill.ID] = skill
	}
	invalid := make([]string, 0)
	for _, id := range pinned {
		skill, ok := byID[id]
		if !ok {
			invalid = append(invalid, id+" (missing)")
			continue
		}
		switch skill.State {
		case catalog.StateConflict, catalog.StateBroken, catalog.StateInvalid:
			invalid = append(invalid, fmt.Sprintf("%s (%s)", id, skill.State))
		}
	}
	if len(invalid) > 0 {
		return Check{
			Name:   "Pinned skills",
			Level:  Error,
			Detail: "invalid: " + strings.Join(invalid, ", "),
		}
	}
	return Check{Name: "Pinned skills", Level: Healthy, Detail: fmt.Sprintf("%d pinned", len(pinned))}
}

func (r *Report) add(section string, check Check) {
	check.Section = section
	r.Checks = append(r.Checks, check)
	if check.Level > r.Overall {
		r.Overall = check.Level
	}
}

type pathProbe struct {
	check    Check
	probe    string
	exists   bool
	writable bool
}

func inspectPath(name, path string) pathProbe {
	probe := pathProbe{check: Check{Name: name}}

	info, err := os.Stat(path)
	if err == nil {
		probe.exists = true
		probe.probe = path
		if !info.IsDir() {
			probe.check.Level = Error
			probe.check.Detail = fmt.Sprintf("%s is not a directory", path)
			return probe
		}

		if err := checkWritable(path); err != nil {
			probe.check.Level = Error
			probe.check.Detail = fmt.Sprintf("%s is not writable: %v", path, err)
			return probe
		}

		probe.writable = true
		probe.check.Level = Healthy
		probe.check.Detail = fmt.Sprintf("%s exists and is writable", path)
		return probe
	}

	if !errors.Is(err, fs.ErrNotExist) {
		probe.check.Level = Error
		probe.check.Detail = fmt.Sprintf("inspect %s: %v", path, err)
		return probe
	}

	parent, parentErr := nearestExistingDir(path)
	if parentErr != nil {
		probe.check.Level = Error
		probe.check.Detail = fmt.Sprintf("%s is missing and cannot be created: %v", path, parentErr)
		return probe
	}

	probe.probe = parent
	if writableErr := checkWritable(parent); writableErr != nil {
		probe.check.Level = Error
		probe.check.Detail = fmt.Sprintf("%s is missing; parent %s is not writable: %v", path, parent, writableErr)
		return probe
	}

	probe.writable = true
	probe.check.Level = Warning
	probe.check.Detail = fmt.Sprintf("%s is missing but can be created", path)
	return probe
}

func nearestExistingDir(path string) (string, error) {
	current, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%s is not a directory", current)
			}
			return current, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no existing parent for %s", path)
		}
		current = parent
	}
}

func checkWritable(dir string) error {
	tempFile, err := os.CreateTemp(dir, ".skiller-doctor-*")
	if err != nil {
		return err
	}

	name := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}

	if err := os.Remove(name); err != nil {
		return err
	}

	return nil
}

func compareVolumes(active, disabled pathProbe) (Level, string) {
	if active.probe == "" || disabled.probe == "" || !active.writable || !disabled.writable {
		return Error, "unable to compare skill directory volumes"
	}

	same, err := sameVolume(active.probe, disabled.probe)
	if err != nil {
		return Error, err.Error()
	}
	if !same {
		return Error, "skill directories are on different volumes"
	}
	if !active.exists || !disabled.exists {
		return Warning, "existing parents are on the same volume"
	}

	return Healthy, "skill directories are on the same volume"
}

func checkRename(active pathProbe) (Level, string) {
	if active.probe == "" || !active.writable {
		return Error, "no writable directory available for rename test"
	}

	source, err := os.MkdirTemp(active.probe, ".skiller-doctor-rename-*")
	if err != nil {
		return Error, fmt.Sprintf("create rename probe: %v", err)
	}
	target, err := os.MkdirTemp(active.probe, ".skiller-doctor-target-*")
	if err != nil {
		_ = os.Remove(source)
		return Error, fmt.Sprintf("create rename target: %v", err)
	}
	if err := os.Remove(target); err != nil {
		_ = os.Remove(source)
		return Error, fmt.Sprintf("prepare rename target: %v", err)
	}

	defer func() {
		_ = os.Remove(source)
		_ = os.Remove(target)
	}()

	if err := os.Rename(source, target); err != nil {
		return Error, fmt.Sprintf("rename probe failed: %v", err)
	}
	if err := os.Rename(target, source); err != nil {
		return Error, fmt.Sprintf("restore rename probe failed: %v", err)
	}

	return Healthy, "directory rename and restore succeeded"
}

func skillCheck(name string, count int) Check {
	return Check{
		Name:   name,
		Level:  Healthy,
		Detail: fmt.Sprintf("%d", count),
	}
}

func issueCheck(name string, count int) Check {
	level := Healthy
	detail := "0"
	if count > 0 {
		level = Error
		detail = fmt.Sprintf("%d found", count)
	}

	return Check{Name: name, Level: level, Detail: detail}
}

func journalCheck(path string) Check {
	check := Check{Name: "Transaction journal"}
	if path == "" {
		check.Level = Error
		check.Detail = "journal path is not configured"
		return check
	}

	_, err := os.Lstat(path)
	switch {
	case err == nil:
		check.Level = Error
		check.Detail = fmt.Sprintf("unfinished journal exists at %s", path)
	case errors.Is(err, fs.ErrNotExist):
		check.Level = Healthy
		check.Detail = "none"
	default:
		check.Level = Error
		check.Detail = fmt.Sprintf("inspect %s: %v", path, err)
	}

	return check
}
