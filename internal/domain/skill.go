package domain

type SkillState string

const (
	SkillStateActive   SkillState = "active"
	SkillStateDisabled SkillState = "disabled"
)

type Skill struct {
	ID    string
	State SkillState
	Path  string
}
