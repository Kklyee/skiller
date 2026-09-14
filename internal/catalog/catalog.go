package catalog

type State uint8

const (
	StateActive State = iota + 1
	StateDisabled
	StateConflict
	StateBroken
	StateInvalid
)

func (s State) String() string {
	switch s {
	case StateActive:
		return "active"
	case StateDisabled:
		return "disabled"
	case StateConflict:
		return "conflict"
	case StateBroken:
		return "broken"
	case StateInvalid:
		return "invalid"
	default:
		return "unknown"
	}
}

type Source uint8

const (
	SourceDirectory Source = iota + 1
	SourceSymlink
	SourceJunction
)

func (s Source) String() string {
	switch s {
	case SourceDirectory:
		return "directory"
	case SourceSymlink:
		return "symlink"
	case SourceJunction:
		return "junction"
	default:
		return "unknown"
	}
}

type Skill struct {
	ID                 string
	State              State
	ActivePath         string
	DisabledPath       string
	ActiveSource       Source
	DisabledSource     Source
	ActiveLinkTarget   string
	DisabledLinkTarget string
	ActiveIssue        string
	DisabledIssue      string
	Name               string
	Description        string
	SkillFile          string
	ActiveSkillFile    string
	DisabledSkillFile  string
}

type Summary struct {
	ActiveDir   string
	DisabledDir string
	Installed   int
	Active      int
	Disabled    int
	Conflict    int
	Broken      int
	Invalid     int
}

func SummaryFor(activeDir, disabledDir string) (Summary, error) {
	skills, err := Scan(activeDir, disabledDir)
	if err != nil {
		return Summary{}, err
	}

	summary := Summarize(skills)
	summary.ActiveDir = activeDir
	summary.DisabledDir = disabledDir

	return summary, nil
}

func Summarize(skills []Skill) Summary {
	summary := Summary{Installed: len(skills)}

	for _, skill := range skills {
		switch skill.State {
		case StateActive:
			summary.Active++
		case StateDisabled:
			summary.Disabled++
		case StateConflict:
			summary.Conflict++
		case StateBroken:
			summary.Broken++
		case StateInvalid:
			summary.Invalid++
		}
	}

	return summary
}
