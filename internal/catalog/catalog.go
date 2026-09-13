package catalog

type State uint8

const (
	StateActive State = iota + 1
	StateDisabled
	StateConflict
)

func (s State) String() string {
	switch s {
	case StateActive:
		return "active"
	case StateDisabled:
		return "disabled"
	case StateConflict:
		return "conflict"
	default:
		return "unknown"
	}
}

type Skill struct {
	ID           string
	State        State
	ActivePath   string
	DisabledPath string
}
