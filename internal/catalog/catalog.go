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
}
