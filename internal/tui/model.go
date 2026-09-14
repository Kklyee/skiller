package tui

import (
	"fmt"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/doctor"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/pin"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/Kklyee/skiller/internal/transaction"
	"github.com/Kklyee/skiller/internal/visibility"
	"slices"
	"strings"
)

type Screen uint8

const (
	ScreenMain Screen = iota + 1
	ScreenGroups
	ScreenGroupEditor
	ScreenGroupDetails
	ScreenDoctor
	ScreenHelp
)

type Focus uint8

const (
	FocusGroups Focus = iota + 1
	FocusSkills
)

type messageKind uint8

const (
	messageInfo messageKind = iota + 1
	messageSuccess
	messageError
)

type Model struct {
	paths paths.Set

	skills  []catalog.Skill
	groups  []group.Group
	pins    []string
	summary catalog.Summary

	screen Screen
	focus  Focus

	selectedGroup  string
	selectedSkill  string
	activeGroup    string
	detailExpanded bool

	search       string
	searchActive bool

	modal       modal
	plan        reconcile.Plan
	input       string
	deleteGroup string

	editorGroup  group.Group
	editorSkills []string
	editorChosen map[string]bool
	editorIndex  int

	doctorReport doctor.Report
	message      string
	messageLevel messageKind
	width        int
	height       int

	startupPending bool
	startupActive  bool
	startupFrame   int
}

const allGroupName = "All"

func NewModel(pathSet paths.Set) (Model, error) {
	model := Model{
		paths:  pathSet,
		screen: ScreenMain,
		focus:  FocusGroups,
		width:  120,
		height: 30,
	}
	if err := model.refresh(); err != nil {
		return Model{}, err
	}

	return model, nil
}

func (m *Model) refresh() error {
	previousGroup := m.selectedGroup
	previousSkill := m.selectedSkill

	skills, err := catalog.Scan(m.paths.Active, m.paths.Disabled)
	if err != nil {
		return err
	}
	groups, err := group.New(m.paths.Groups).ListWithMissing(skills)
	if err != nil {
		return err
	}
	pinned, err := pin.New(m.paths.Pins).List()
	if err != nil {
		return err
	}

	m.skills = skills
	m.groups = groups
	m.pins = pinned
	m.activeGroup = findActiveGroup(groups, skills)
	m.summary = catalog.Summarize(skills)
	m.summary.ActiveDir = m.paths.Active
	m.summary.DisabledDir = m.paths.Disabled
	m.selectedGroup = previousGroup
	m.selectedSkill = previousSkill
	m.normalizeGroupSelection()
	m.normalizeSelection()

	return nil
}

func (m *Model) moveSelection(delta int) {
	if m.focus == FocusGroups {
		m.moveGroup(delta)
		return
	}
	if m.focus != FocusSkills {
		return
	}

	visible := m.visibleSkills()
	if len(visible) == 0 {
		m.selectedSkill = ""
		return
	}
	index := 0
	for i, skill := range visible {
		if skill.ID == m.selectedSkill {
			index = i
			break
		}
	}
	index += delta
	if index < 0 {
		index = 0
	}
	if index >= len(visible) {
		index = len(visible) - 1
	}
	m.selectedSkill = visible[index].ID
}

func (m *Model) moveGroup(delta int) {
	if len(m.groups) == 0 {
		m.selectedGroup = ""
		return
	}
	index := -1
	if m.selectedGroup != "" {
		for i, group := range m.groups {
			if group.Name == m.selectedGroup {
				index = i
				break
			}
		}
	}
	if m.screen == ScreenGroups {
		if index < 0 {
			index = 0
		} else {
			index += delta
		}
	} else {
		index += delta
	}

	if m.screen == ScreenGroups && index < 0 {
		index = 0
	}
	if index < -1 {
		index = -1
	}
	if index >= len(m.groups) {
		index = len(m.groups) - 1
	}
	if index == -1 {
		m.selectedGroup = ""
		return
	}
	m.selectedGroup = m.groups[index].Name
	m.normalizeSelection()
}

func (m *Model) normalizeSelection() {
	visible := m.visibleSkills()
	if len(visible) == 0 {
		m.selectedSkill = ""
		return
	}
	for _, skill := range visible {
		if skill.ID == m.selectedSkill {
			return
		}
	}
	m.selectedSkill = visible[0].ID
}

func (m *Model) normalizeGroupSelection() {
	if len(m.groups) == 0 {
		m.selectedGroup = ""
		return
	}
	for _, group := range m.groups {
		if group.Name == m.selectedGroup {
			return
		}
	}
	if m.screen == ScreenGroups {
		m.selectedGroup = m.groups[0].Name
	}
}

func (m *Model) visibleSkills() []catalog.Skill {
	groupIDs := map[string]bool(nil)
	if m.selectedGroup != "" {
		groupIDs = make(map[string]bool)
		for _, group := range m.groups {
			if group.Name == m.selectedGroup {
				for _, id := range group.Skills {
					groupIDs[id] = true
				}
				break
			}
		}
	}

	query := strings.ToLower(m.search)
	visible := make([]catalog.Skill, 0, len(m.skills))
	for _, skill := range m.skills {
		if groupIDs != nil && !groupIDs[skill.ID] {
			continue
		}
		if query != "" && !m.matches(skill, query) {
			continue
		}
		visible = append(visible, skill)
	}

	return visible
}

func (m *Model) matches(skill catalog.Skill, query string) bool {
	values := []string{skill.ID, skill.Name, skill.Description}
	for _, group := range m.groups {
		for _, id := range group.Skills {
			if id == skill.ID {
				values = append(values, group.Name)
				break
			}
		}
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func (m *Model) selectedSkillValue() (catalog.Skill, bool) {
	for _, skill := range m.visibleSkills() {
		if skill.ID == m.selectedSkill {
			return skill, true
		}
	}
	return catalog.Skill{}, false
}

func (m *Model) isPinned(id string) bool {
	return slices.Contains(m.pins, id)
}

func (m *Model) toggleSelectedSkill() {
	skill, ok := m.selectedSkillValue()
	if !ok {
		return
	}

	var err error
	switch skill.State {
	case catalog.StateActive:
		if m.isPinned(skill.ID) {
			m.setMessage(messageInfo, fmt.Sprintf("Pinned skill %s stays active; unpin it first", skill.ID))
			return
		}
		_, err = visibility.Disable(m.paths.Active, m.paths.Disabled, skill.ID)
	case catalog.StateDisabled:
		_, err = visibility.Enable(m.paths.Active, m.paths.Disabled, skill.ID)
	default:
		err = fmt.Errorf("cannot toggle %s skill %q", skill.State, skill.ID)
	}
	if err != nil {
		m.setError(err)
		return
	}
	if err := m.refresh(); err != nil {
		m.setError(err)
		return
	}
	m.setMessage(messageSuccess, fmt.Sprintf("Toggled %s", skill.ID))
}

func (m *Model) toggleAllSkills() {
	skills := m.visibleSkills()
	if len(skills) == 0 {
		return
	}

	allActive := true
	desired := make([]string, 0, len(skills))
	for _, skill := range skills {
		desired = append(desired, skill.ID)
		if skill.State != catalog.StateActive {
			allActive = false
		}
	}
	if allActive {
		desired = nil
	}

	plan := reconcile.BuildWithPins(group.Group{Name: "all skills", Skills: desired}, skills, m.pins)
	if plan.HasIssues() {
		m.setMessage(messageError, "Cannot toggle all skills: resolve catalog issues first")
		return
	}
	if plan.Changes() == 0 {
		return
	}
	if err := transaction.Apply(m.paths, plan); err != nil {
		m.setError(err)
		return
	}
	if err := m.refresh(); err != nil {
		m.setError(err)
		return
	}

	if allActive {
		m.setMessage(messageSuccess, "Disabled all visible skills")
	} else {
		m.setMessage(messageSuccess, "Activated all visible skills")
	}
}

func (m *Model) openReconcile() {
	if m.selectedGroup == "" {
		selected := group.Group{Name: allGroupName, Skills: make([]string, 0, len(m.skills))}
		for _, skill := range m.skills {
			selected.Skills = append(selected.Skills, skill.ID)
		}
		m.plan = reconcile.BuildWithPins(selected, m.skills, m.pins)
		m.modal = modalReconcile
		return
	}
	for _, group := range m.groups {
		if group.Name == m.selectedGroup {
			m.plan = reconcile.BuildWithPins(group, m.skills, m.pins)
			m.modal = modalReconcile
			return
		}
	}
}

func (m *Model) openEditor() {
	if m.selectedGroup == "" {
		m.setMessage(messageInfo, "Select a group before editing")
		return
	}
	for _, group := range m.groups {
		if group.Name != m.selectedGroup {
			continue
		}
		m.editorGroup = group
		m.editorSkills = make([]string, 0, len(m.skills))
		for _, skill := range m.skills {
			m.editorSkills = append(m.editorSkills, skill.ID)
		}
		known := make(map[string]bool, len(m.editorSkills))
		for _, id := range m.editorSkills {
			known[id] = true
		}
		for _, id := range group.Skills {
			if !known[id] {
				m.editorSkills = append(m.editorSkills, id)
			}
		}
		slices.Sort(m.editorSkills)
		m.editorChosen = make(map[string]bool, len(group.Skills))
		for _, id := range group.Skills {
			m.editorChosen[id] = true
		}
		m.editorIndex = 0
		m.screen = ScreenGroupEditor
		return
	}
}

func (m *Model) saveEditor() {
	store := group.New(m.paths.Groups)
	selected := make([]string, 0)
	for _, id := range m.editorSkills {
		if m.editorChosen[id] {
			selected = append(selected, id)
		}
	}

	if len(m.editorGroup.Skills) > 0 {
		if _, err := store.Remove(m.editorGroup.Name, m.editorGroup.Skills...); err != nil {
			m.setError(err)
			return
		}
	}
	if len(selected) > 0 {
		if _, err := store.Add(m.editorGroup.Name, selected...); err != nil {
			m.setError(err)
			return
		}
	}
	if err := m.refresh(); err != nil {
		m.setError(err)
		return
	}
	m.screen = ScreenGroups
	m.setMessage(messageSuccess, fmt.Sprintf("Saved group %s", m.editorGroup.Name))
}

func (m *Model) openDeleteGroup() {
	if m.selectedGroup == "" {
		m.setMessage(messageInfo, "Select a group before deleting")
		return
	}
	m.deleteGroup = m.selectedGroup
	m.modal = modalDeleteGroup
}

func findActiveGroup(groups []group.Group, skills []catalog.Skill) string {
	active := make(map[string]struct{})
	for _, skill := range skills {
		switch skill.State {
		case catalog.StateActive:
			active[skill.ID] = struct{}{}
		case catalog.StateConflict, catalog.StateBroken, catalog.StateInvalid:
			return ""
		}
	}

	matches := make([]string, 0, 1)
	for _, candidate := range groups {
		if len(candidate.Missing) > 0 || len(candidate.Skills) != len(active) {
			continue
		}
		matched := true
		for _, id := range candidate.Skills {
			if _, ok := active[id]; !ok {
				matched = false
				break
			}
		}
		if matched {
			matches = append(matches, candidate.Name)
		}
	}
	if len(matches) != 1 {
		return ""
	}
	return matches[0]
}

func (m *Model) setMessage(kind messageKind, message string) {
	m.message = message
	m.messageLevel = kind
}

func (m *Model) setError(err error) {
	m.setMessage(messageError, err.Error())
}

func (m *Model) clearMessage() {
	m.message = ""
	m.messageLevel = 0
}
