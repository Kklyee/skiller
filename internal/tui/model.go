package tui

import (
	"fmt"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/doctor"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/pin"
	"github.com/Kklyee/skiller/internal/profile"
	"github.com/Kklyee/skiller/internal/project"
	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
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
	ScreenProfiles
	ScreenProfileEditor
	ScreenProject
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

	skills     []catalog.Skill
	groups     []group.Group
	profiles   []profile.Profile
	pins       []string
	provenance map[string]skillprovenance.Entry
	summary    catalog.Summary

	projectConfig project.Config
	projectPath   string
	projectLoaded bool
	projectError  string

	screen Screen
	focus  Focus

	selectedGroup   string
	selectedProfile string
	selectedSkill   string
	selectedSkills  map[string]bool
	activeGroup     string
	detailExpanded  bool

	search       string
	searchActive bool

	paletteQuery string
	paletteIndex int

	modal         modal
	plan          reconcile.Plan
	planKind      string
	input         string
	deleteGroup   string
	deleteProfile string

	editorGroup  group.Group
	editorSkills []string
	editorChosen map[string]bool
	editorIndex  int

	profileEditor        profile.Profile
	profileEditorGroups  map[string]bool
	profileEditorSkills  map[string]bool
	profileEditorExclude map[string]bool
	profileEditorSection int
	profileEditorIndex   int

	doctorReport doctor.Report
	message      string
	messageLevel messageKind
	width        int
	height       int

	startupPending bool
	startupActive  bool
	startupFrame   int

	batchAction     batchAction
	batchGroupIndex int
}

const allGroupName = "All"

const (
	profileEditorGroups = iota
	profileEditorSkills
	profileEditorExclude
)

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
	previousProfile := m.selectedProfile
	previousSkill := m.selectedSkill

	skills, err := catalog.Scan(m.paths.Active, m.paths.Disabled)
	if err != nil {
		return err
	}
	groups, err := group.New(m.paths.Groups).ListWithMissing(skills)
	if err != nil {
		return err
	}
	profiles, err := profile.New(m.paths.Profiles).List()
	if err != nil {
		return err
	}
	pinned, err := pin.New(m.paths.Pins).List()
	if err != nil {
		return err
	}
	provenanceData, err := skillprovenance.New(m.paths.Provenance).List()
	if err != nil {
		return err
	}

	m.skills = skills
	m.groups = groups
	m.profiles = profiles
	m.pins = pinned
	m.provenance = provenanceData
	m.activeGroup = findActiveGroup(groups, skills, pinned)
	m.summary = catalog.Summarize(skills)
	m.summary.ActiveDir = m.paths.Active
	m.summary.DisabledDir = m.paths.Disabled
	m.selectedGroup = previousGroup
	m.selectedProfile = previousProfile
	m.selectedSkill = previousSkill
	m.normalizeSkillSelection()
	m.normalizeGroupSelection()
	m.normalizeProfileSelection()
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
	previous := m.selectedGroup
	if len(m.groups) == 0 {
		m.selectedGroup = ""
		if previous != m.selectedGroup {
			m.clearSelectedSkills()
		}
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
		if previous != m.selectedGroup {
			m.clearSelectedSkills()
		}
		return
	}
	m.selectedGroup = m.groups[index].Name
	if previous != m.selectedGroup {
		m.clearSelectedSkills()
	}
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

func (m *Model) normalizeProfileSelection() {
	for _, stored := range m.profiles {
		if stored.Name == m.selectedProfile {
			return
		}
	}
	if len(m.profiles) == 0 {
		m.selectedProfile = ""
		return
	}
	m.selectedProfile = m.profiles[0].Name
}

func (m *Model) moveProfile(delta int) {
	if len(m.profiles) == 0 {
		m.selectedProfile = ""
		return
	}
	index := 0
	for i, stored := range m.profiles {
		if stored.Name == m.selectedProfile {
			index = i
			break
		}
	}
	index += delta
	if index < 0 {
		index = 0
	}
	if index >= len(m.profiles) {
		index = len(m.profiles) - 1
	}
	m.selectedProfile = m.profiles[index].Name
}

func (m *Model) selectedProfileValue() (profile.Profile, bool) {
	for _, stored := range m.profiles {
		if stored.Name == m.selectedProfile {
			return stored, true
		}
	}
	return profile.Profile{}, false
}

func (m *Model) openProfileName() {
	m.input = ""
	m.modal = modalProfileName
	m.clearMessage()
}

func (m *Model) openProfileEditor() {
	stored, ok := m.selectedProfileValue()
	if !ok {
		m.setMessage(messageInfo, "Select a profile before editing")
		return
	}
	m.profileEditor = stored
	m.profileEditorGroups = profileEditorSelection(stored.Groups)
	m.profileEditorSkills = profileEditorSelection(stored.Skills)
	m.profileEditorExclude = profileEditorSelection(stored.Exclude)
	m.profileEditorSection = profileEditorGroups
	m.profileEditorIndex = 0
	m.screen = ScreenProfileEditor
}

func (m *Model) openDeleteProfile() {
	if m.selectedProfile == "" {
		m.setMessage(messageInfo, "Select a profile before deleting")
		return
	}
	m.deleteProfile = m.selectedProfile
	m.modal = modalDeleteProfile
}

func profileEditorSelection(values []string) map[string]bool {
	selected := make(map[string]bool, len(values))
	for _, value := range values {
		selected[value] = true
	}
	return selected
}

func (m *Model) profileEditorItems() []string {
	return m.profileEditorItemsFor(m.profileEditorSection)
}

func (m *Model) profileEditorItemsFor(section int) []string {
	selected := m.profileEditorGroups
	items := make([]string, 0)
	if section == profileEditorGroups {
		items = make([]string, 0, len(m.groups)+len(selected))
		for _, current := range m.groups {
			items = append(items, current.Name)
		}
	} else {
		if section == profileEditorSkills {
			selected = m.profileEditorSkills
		} else {
			selected = m.profileEditorExclude
		}
		items = make([]string, 0, len(m.skills)+len(selected))
		for _, skill := range m.skills {
			items = append(items, skill.ID)
		}
	}
	known := make(map[string]struct{}, len(items))
	for _, item := range items {
		known[item] = struct{}{}
	}
	for item := range selected {
		if _, ok := known[item]; ok {
			continue
		}
		items = append(items, item)
	}
	slices.Sort(items)
	return items
}

func (m *Model) profileEditorChosen(id string) bool {
	switch m.profileEditorSection {
	case profileEditorGroups:
		return m.profileEditorGroups[id]
	case profileEditorSkills:
		return m.profileEditorSkills[id]
	case profileEditorExclude:
		return m.profileEditorExclude[id]
	default:
		return false
	}
}

func (m *Model) toggleProfileEditorSelection() {
	items := m.profileEditorItems()
	if len(items) == 0 || m.profileEditorIndex < 0 || m.profileEditorIndex >= len(items) {
		return
	}
	id := items[m.profileEditorIndex]
	selected := m.profileEditorGroups
	if m.profileEditorSection == profileEditorSkills {
		selected = m.profileEditorSkills
	} else if m.profileEditorSection == profileEditorExclude {
		selected = m.profileEditorExclude
	}
	selected[id] = !selected[id]
	if !selected[id] {
		delete(selected, id)
	}
}

func (m *Model) selectAllProfileEditorItems() {
	items := m.profileEditorItems()
	for _, id := range items {
		selected := m.profileEditorGroups
		if m.profileEditorSection == profileEditorSkills {
			selected = m.profileEditorSkills
		} else if m.profileEditorSection == profileEditorExclude {
			selected = m.profileEditorExclude
		}
		selected[id] = true
	}
}

func (m *Model) moveProfileEditor(delta int) {
	items := m.profileEditorItems()
	if len(items) == 0 {
		m.profileEditorIndex = 0
		return
	}
	m.profileEditorIndex += delta
	if m.profileEditorIndex < 0 {
		m.profileEditorIndex = 0
	}
	if m.profileEditorIndex >= len(items) {
		m.profileEditorIndex = len(items) - 1
	}
}

func (m *Model) changeProfileEditorSection(delta int) {
	m.profileEditorSection += delta
	if m.profileEditorSection < profileEditorGroups {
		m.profileEditorSection = profileEditorExclude
	}
	if m.profileEditorSection > profileEditorExclude {
		m.profileEditorSection = profileEditorGroups
	}
	m.profileEditorIndex = 0
}

func (m *Model) profileEditorValues(section int) []string {
	items := m.profileEditorItemsFor(section)
	values := make([]string, 0, len(items))
	for _, id := range items {
		selected := m.profileEditorGroups
		if section == profileEditorSkills {
			selected = m.profileEditorSkills
		} else if section == profileEditorExclude {
			selected = m.profileEditorExclude
		}
		if selected[id] {
			values = append(values, id)
		}
	}
	return values
}

func (m *Model) saveProfileEditor() {
	updated := profile.Profile{
		Name:    m.profileEditor.Name,
		Groups:  m.profileEditorValues(profileEditorGroups),
		Skills:  m.profileEditorValues(profileEditorSkills),
		Exclude: m.profileEditorValues(profileEditorExclude),
	}
	if err := profile.New(m.paths.Profiles).Update(updated.Name, updated); err != nil {
		m.setError(err)
		return
	}
	if err := m.refresh(); err != nil {
		m.setError(err)
		return
	}
	m.selectedProfile = updated.Name
	m.screen = ScreenProfiles
	m.setMessage(messageSuccess, fmt.Sprintf("Saved profile %s", updated.Name))
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

func (m *Model) selectedSkillCount() int {
	return len(m.selectedSkills)
}

func (m *Model) toggleSelectedMark() {
	if _, ok := m.selectedSkillValue(); !ok {
		return
	}
	if m.selectedSkills == nil {
		m.selectedSkills = make(map[string]bool)
	}
	m.selectedSkills[m.selectedSkill] = !m.selectedSkills[m.selectedSkill]
	if !m.selectedSkills[m.selectedSkill] {
		delete(m.selectedSkills, m.selectedSkill)
	}
}

func (m *Model) clearSelectedSkills() {
	m.selectedSkills = nil
}

func (m *Model) normalizeSkillSelection() {
	if len(m.selectedSkills) == 0 {
		return
	}
	known := make(map[string]struct{}, len(m.skills))
	for _, skill := range m.skills {
		known[skill.ID] = struct{}{}
	}
	for id := range m.selectedSkills {
		if _, ok := known[id]; !ok {
			delete(m.selectedSkills, id)
		}
	}
}

func (m *Model) selectedSkillIDs() []string {
	ids := make([]string, 0, len(m.selectedSkills))
	for id, selected := range m.selectedSkills {
		if selected {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}

func (m *Model) openBatchActions() {
	if m.selectedSkillCount() == 0 {
		m.setMessage(messageInfo, "Mark skills before opening batch actions")
		return
	}
	m.batchAction = batchActionNone
	m.batchGroupIndex = 0
	m.modal = modalBatch
}

func (m *Model) applyBatchVisibility(enable bool) {
	ids := m.selectedSkillIDs()
	if len(ids) == 0 {
		m.modal = modalNone
		return
	}
	selected := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		selected[id] = struct{}{}
	}
	desired := make([]string, 0, len(m.skills))
	for _, skill := range m.skills {
		wantActive := skill.State == catalog.StateActive
		if _, ok := selected[skill.ID]; ok {
			wantActive = enable
		}
		if wantActive {
			desired = append(desired, skill.ID)
		}
	}

	plan := reconcile.BuildWithPins(group.Group{Name: "selection", Skills: desired}, m.skills, m.pins)
	if plan.HasIssues() {
		m.setMessage(messageError, "Cannot apply batch action: resolve catalog or pin issues first")
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
	m.modal = modalNone
	m.clearSelectedSkills()
	action := "Disabled"
	if enable {
		action = "Enabled"
	}
	m.setMessage(messageSuccess, fmt.Sprintf("%s %d selected skills", action, len(ids)))
}

func (m *Model) openBatchGroup(action batchAction) {
	if len(m.groups) == 0 {
		m.setMessage(messageInfo, "Create a group before assigning skills")
		return
	}
	m.batchAction = action
	m.batchGroupIndex = 0
	for index, candidate := range m.groups {
		if candidate.Name == m.selectedGroup {
			m.batchGroupIndex = index
			break
		}
	}
	m.modal = modalBatchGroup
}

func (m *Model) moveBatchGroup(delta int) {
	if len(m.groups) == 0 {
		return
	}
	m.batchGroupIndex += delta
	if m.batchGroupIndex < 0 {
		m.batchGroupIndex = 0
	}
	if m.batchGroupIndex >= len(m.groups) {
		m.batchGroupIndex = len(m.groups) - 1
	}
}

func (m *Model) applyBatchGroup() {
	ids := m.selectedSkillIDs()
	if len(ids) == 0 || m.batchGroupIndex < 0 || m.batchGroupIndex >= len(m.groups) {
		m.modal = modalNone
		return
	}
	selectedGroup := m.groups[m.batchGroupIndex]
	blocked := make([]string, 0)
	if m.batchAction == batchActionRemoveGroup {
		removable := make([]string, 0, len(ids))
		for _, id := range ids {
			if m.isPinned(id) {
				blocked = append(blocked, id)
				continue
			}
			removable = append(removable, id)
		}
		ids = removable
		if len(ids) == 0 {
			m.modal = modalNone
			m.clearSelectedSkills()
			m.setMessage(messageInfo, "Pinned skills stay in every group")
			return
		}
	}
	store := group.New(m.paths.Groups)
	count := 0
	var err error
	if m.batchAction == batchActionAddGroup {
		count, err = store.Add(selectedGroup.Name, ids...)
	} else {
		count, err = store.Remove(selectedGroup.Name, ids...)
	}
	if err != nil {
		m.setError(err)
		return
	}
	if err := m.refresh(); err != nil {
		m.setError(err)
		return
	}
	m.modal = modalNone
	m.clearSelectedSkills()
	action := "Removed"
	preposition := "from"
	if m.batchAction == batchActionAddGroup {
		action = "Added"
		preposition = "to"
	}
	message := fmt.Sprintf("%s %d skills %s %s", action, count, preposition, selectedGroup.Name)
	if len(blocked) > 0 {
		message += "; pinned skills kept: " + strings.Join(blocked, ", ")
	}
	m.setMessage(messageSuccess, message)
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
	m.planKind = "group"
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

func (m *Model) openProfileReconcile() {
	stored, ok := m.selectedProfileValue()
	if !ok {
		m.setMessage(messageInfo, "Select a profile before using it")
		return
	}
	target := profile.Resolve(stored, m.groups)
	m.plan = reconcile.BuildWithPins(target.Group, m.skills, m.pins)
	for _, name := range target.MissingGroups {
		m.plan.Issues = append(m.plan.Issues, fmt.Sprintf("missing group %s", name))
	}
	slices.Sort(m.plan.Issues)
	m.planKind = "profile"
	m.modal = modalReconcile
}

func (m *Model) openProject() {
	m.projectConfig = project.Config{}
	m.projectPath = ""
	m.projectLoaded = false
	m.projectError = ""
	config, path, err := project.Load(".")
	if err != nil {
		m.projectError = err.Error()
	} else {
		m.projectConfig = config
		m.projectPath = path
		m.projectLoaded = true
	}
	m.screen = ScreenProject
	m.clearMessage()
}

func (m *Model) projectTarget() (group.Group, []string, error) {
	if !m.projectLoaded {
		return group.Group{}, nil, fmt.Errorf("project config is not loaded")
	}
	config := m.projectConfig
	base := group.Group{Name: "project", Skills: config.Skills}
	missingGroups := []string(nil)
	if config.Profile != "" {
		stored, ok := m.profileByName(config.Profile)
		if !ok {
			return group.Group{}, nil, fmt.Errorf("profile %q does not exist", config.Profile)
		}
		target := profile.Resolve(stored, m.groups)
		base = target.Group
		missingGroups = target.MissingGroups
	}
	base = project.ApplyOverrides(base, config.Include, config.Exclude)
	base.Name = "project"
	return base, missingGroups, nil
}

func (m *Model) profileByName(name string) (profile.Profile, bool) {
	for _, stored := range m.profiles {
		if stored.Name == name {
			return stored, true
		}
	}
	return profile.Profile{}, false
}

func (m *Model) openProjectReconcile() {
	target, missingGroups, err := m.projectTarget()
	if err != nil {
		m.setMessage(messageError, err.Error())
		return
	}
	m.plan = reconcile.BuildWithPins(target, m.skills, m.pins)
	for _, name := range missingGroups {
		m.plan.Issues = append(m.plan.Issues, fmt.Sprintf("missing group %s", name))
	}
	slices.Sort(m.plan.Issues)
	m.planKind = "project"
	m.modal = modalReconcile
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
		for _, id := range m.pins {
			if known[id] {
				m.editorChosen[id] = true
			}
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

	removable := make([]string, 0, len(m.editorGroup.Skills))
	for _, id := range m.editorGroup.Skills {
		if !m.isPinned(id) {
			removable = append(removable, id)
		}
	}
	if len(removable) > 0 {
		if _, err := store.Remove(m.editorGroup.Name, removable...); err != nil {
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

func findActiveGroup(groups []group.Group, skills []catalog.Skill, pinned []string) string {
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
		if len(candidate.Missing) > 0 {
			continue
		}
		desired := make(map[string]struct{}, len(candidate.Skills)+len(pinned))
		for _, id := range candidate.Skills {
			desired[id] = struct{}{}
		}
		for _, id := range pinned {
			desired[id] = struct{}{}
		}
		if len(desired) != len(active) {
			continue
		}
		matched := true
		for id := range desired {
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
