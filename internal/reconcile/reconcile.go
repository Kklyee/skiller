package reconcile

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/visibility"
)

type Plan struct {
	Group        string
	Enable       []string
	Disable      []string
	Keep         []string
	KeepActive   []string
	KeepDisabled []string
	Missing      []string
	Issues       []string
}

func Build(selected group.Group, skills []catalog.Skill) Plan {
	return BuildWithPins(selected, skills, nil)
}

func BuildWithPins(selected group.Group, skills []catalog.Skill, pinned []string) Plan {
	plan := Plan{Group: selected.Name}
	desired := make(map[string]struct{}, len(selected.Skills)+len(pinned))
	for _, skill := range selected.Skills {
		desired[skill] = struct{}{}
	}
	for _, skill := range pinned {
		desired[skill] = struct{}{}
	}

	known := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		known[skill.ID] = struct{}{}
		_, wanted := desired[skill.ID]

		switch skill.State {
		case catalog.StateActive:
			if wanted {
				plan.Keep = append(plan.Keep, skill.ID)
				plan.KeepActive = append(plan.KeepActive, skill.ID)
			} else {
				plan.Disable = append(plan.Disable, skill.ID)
			}
		case catalog.StateDisabled:
			if wanted {
				plan.Enable = append(plan.Enable, skill.ID)
			} else {
				plan.Keep = append(plan.Keep, skill.ID)
				plan.KeepDisabled = append(plan.KeepDisabled, skill.ID)
			}
		case catalog.StateConflict, catalog.StateBroken, catalog.StateInvalid:
			plan.Issues = append(plan.Issues, issueText(skill, wanted))
		}
	}

	requested := make(map[string]struct{}, len(selected.Skills)+len(pinned))
	for _, skill := range selected.Skills {
		requested[skill] = struct{}{}
	}
	for _, skill := range pinned {
		requested[skill] = struct{}{}
	}
	for skill := range requested {
		if _, ok := known[skill]; !ok {
			plan.Missing = append(plan.Missing, skill)
		}
	}

	slices.Sort(plan.Enable)
	slices.Sort(plan.Disable)
	slices.Sort(plan.Keep)
	slices.Sort(plan.KeepActive)
	slices.Sort(plan.KeepDisabled)
	slices.Sort(plan.Missing)
	slices.Sort(plan.Issues)

	return plan
}

func (p Plan) Changes() int {
	return len(p.Enable) + len(p.Disable)
}

func (p Plan) HasIssues() bool {
	return len(p.Missing) > 0 || len(p.Issues) > 0
}

func Apply(activeDir, disabledDir string, plan Plan) error {
	if plan.HasIssues() {
		return errors.New("cannot apply plan with missing skills or catalog issues")
	}

	for _, id := range plan.Enable {
		if _, err := visibility.Enable(activeDir, disabledDir, id); err != nil {
			return fmt.Errorf("enable %q: %w", id, err)
		}
	}
	for _, id := range plan.Disable {
		if _, err := visibility.Disable(activeDir, disabledDir, id); err != nil {
			return fmt.Errorf("disable %q: %w", id, err)
		}
	}

	return nil
}

func issueText(skill catalog.Skill, wanted bool) string {
	action := "keep"
	if wanted {
		action = "enable"
	}

	detail := skill.State.String()
	if skill.ActiveIssue != "" {
		detail = skill.ActiveIssue
	}
	if skill.DisabledIssue != "" {
		detail = skill.DisabledIssue
	}

	return fmt.Sprintf("%s %s: %s", action, skill.ID, detail)
}
