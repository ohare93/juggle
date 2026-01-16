package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
)

// StandaloneEditModel is a TUI model for standalone ball editing (from CLI edit command)
// It exits after saving or cancelling
type StandaloneEditModel struct {
	store        *session.Store
	sessionStore *session.SessionStore
	sessions     []*session.JuggleSession
	ball         *session.Ball // The ball being edited

	// Form state
	textInput                 textinput.Model
	contextInput              textarea.Model
	pendingBallContext        string
	pendingBallIntent         string
	pendingBallPriority       int      // Index in priority options (0=low, 1=medium, 2=high, 3=urgent)
	pendingBallTags           string   // Comma-separated tags
	pendingBallSession        int      // Index in session options (0=none, 1+ = session index)
	pendingBallModelSize      int      // Index in model size options (0=default, 1=small, 2=medium, 3=large)
	pendingBallDependsOn      []string // Selected dependency ball IDs
	pendingBallBlockingReason int      // Index in blocking reason options (0=blank, 1=Human needed, 2=Waiting for dependency, 3=Needs research, 4=custom)
	pendingBallCustomReason   string   // Custom blocking reason text (when pendingBallBlockingReason == 4)
	pendingBallFormField      int      // Current field in form
	pendingAcceptanceCriteria []string // Acceptance criteria being collected
	pendingNewAC              string   // Content of the "new AC" field, preserved during navigation

	// File autocomplete state
	fileAutocomplete *AutocompleteState

	// Dependency selector state
	dependencySelectBalls  []*session.Ball // Non-complete balls available for selection
	dependencySelectIndex  int             // Current selection index in dependency selector
	dependencySelectActive map[string]bool // Which dependencies are currently selected (by ID)
	inDependencySelector   bool            // Whether we're in dependency selector mode

	// UI state
	width   int
	height  int
	message string
	done    bool          // True when form is completed (either save or cancel)
	result  *session.Ball // The updated ball (nil if cancelled)
	err     error
}

// StandaloneEditResult contains the result of the standalone ball editing
type StandaloneEditResult struct {
	Ball      *session.Ball // Updated ball, nil if cancelled
	Cancelled bool          // True if user cancelled
	Err       error         // Error if any
}

// NewStandaloneEditModel creates a new standalone ball editing model
func NewStandaloneEditModel(store *session.Store, sessionStore *session.SessionStore, ball *session.Ball) StandaloneEditModel {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 60
	ti.Placeholder = "What is this ball about? (50 char recommended)"
	ti.Blur() // Start with context field focused, not this

	ta := textarea.New()
	ta.Placeholder = "Background context for this task"
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.Focus() // Context field is first, so focus it

	// Map ball priority to index
	priorityIdx := 1 // default medium
	switch ball.Priority {
	case session.PriorityLow:
		priorityIdx = 0
	case session.PriorityMedium:
		priorityIdx = 1
	case session.PriorityHigh:
		priorityIdx = 2
	case session.PriorityUrgent:
		priorityIdx = 3
	}

	// Map ball model size to index
	modelSizeIdx := 0 // default blank
	switch ball.ModelSize {
	case session.ModelSizeBlank:
		modelSizeIdx = 0
	case session.ModelSizeSmall:
		modelSizeIdx = 1
	case session.ModelSizeMedium:
		modelSizeIdx = 2
	case session.ModelSizeLarge:
		modelSizeIdx = 3
	}

	// Map blocking reason
	blockingReasonIdx := 0
	customReason := ""
	switch ball.BlockedReason {
	case "":
		blockingReasonIdx = 0
	case "Human needed":
		blockingReasonIdx = 1
	case "Waiting for dependency":
		blockingReasonIdx = 2
	case "Needs research":
		blockingReasonIdx = 3
	default:
		blockingReasonIdx = 4
		customReason = ball.BlockedReason
	}

	// Set context in textarea
	ta.SetValue(ball.Context)

	m := StandaloneEditModel{
		store:                     store,
		sessionStore:              sessionStore,
		ball:                      ball,
		textInput:                 ti,
		contextInput:              ta,
		pendingBallContext:        ball.Context,
		pendingBallIntent:         ball.Title,
		pendingBallPriority:       priorityIdx,
		pendingBallTags:           strings.Join(ball.Tags, ", "),
		pendingBallModelSize:      modelSizeIdx,
		pendingBallDependsOn:      ball.DependsOn,
		pendingBallBlockingReason: blockingReasonIdx,
		pendingBallCustomReason:   customReason,
		pendingAcceptanceCriteria: make([]string, len(ball.AcceptanceCriteria)),
		fileAutocomplete:          NewAutocompleteState(store.ProjectDir()),
	}
	copy(m.pendingAcceptanceCriteria, ball.AcceptanceCriteria)

	adjustStandaloneEditContextHeight(&m)

	return m
}

func (m StandaloneEditModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		loadSessionsForStandaloneEdit(m.sessionStore),
	)
}

type loadedSessionsForStandaloneEditMsg struct {
	sessions []*session.JuggleSession
	err      error
}

func loadSessionsForStandaloneEdit(sessionStore *session.SessionStore) tea.Cmd {
	return func() tea.Msg {
		if sessionStore == nil {
			return loadedSessionsForStandaloneEditMsg{sessions: nil}
		}
		sessions, err := sessionStore.ListSessions()
		if err != nil {
			return loadedSessionsForStandaloneEditMsg{err: err}
		}
		return loadedSessionsForStandaloneEditMsg{sessions: sessions}
	}
}

func (m StandaloneEditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedSessionsForStandaloneEditMsg:
		if msg.err != nil {
			m.message = "Warning: couldn't load sessions: " + msg.err.Error()
		} else {
			m.sessions = msg.sessions
			// Find the session that matches the ball's tags
			for i, sess := range m.sessions {
				if sess.ID == PseudoSessionAll || sess.ID == PseudoSessionUntagged {
					continue
				}
				for _, tag := range m.ball.Tags {
					if tag == sess.ID {
						// Account for "(none)" being index 0
						realIdx := 1
						for _, s := range m.sessions {
							if s.ID == PseudoSessionAll || s.ID == PseudoSessionUntagged {
								continue
							}
							if s.ID == sess.ID {
								m.pendingBallSession = realIdx
								break
							}
							realIdx++
						}
						break
					}
				}
				_ = i // silence unused warning
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.inDependencySelector {
			return m.handleDependencySelectorKey(msg)
		}
		return m.handleFormKey(msg)
	}

	return m, nil
}

func (m StandaloneEditModel) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Field indices are dynamic due to variable AC count
	// Order: Context(0), Title(1), ACs(2 to 2+len(ACs)), Tags, Session, ModelSize, Priority, BlockingReason, DependsOn, Save
	const (
		fieldContext = 0
		fieldIntent  = 1
		fieldACStart = 2
	)
	fieldACEnd := fieldACStart + len(m.pendingAcceptanceCriteria)
	fieldTags := fieldACEnd + 1
	fieldSession := fieldTags + 1
	fieldModelSize := fieldSession + 1
	fieldPriority := fieldModelSize + 1
	fieldBlockingReason := fieldPriority + 1
	fieldDependsOn := fieldBlockingReason + 1
	fieldSave := fieldDependsOn + 1

	numModelSizeOptions := 4
	numPriorityOptions := 4
	numBlockingReasonOptions := 5
	numSessionOptions := 1
	for _, sess := range m.sessions {
		if sess.ID != PseudoSessionAll && sess.ID != PseudoSessionUntagged {
			numSessionOptions++
		}
	}
	maxFieldIndex := fieldSave

	isTextInputField := func(field int) bool {
		// Blocking reason field is text input only when custom option (4) is selected
		if field == fieldBlockingReason && m.pendingBallBlockingReason == 4 {
			return true
		}
		return field == fieldContext || field == fieldIntent || field == fieldTags ||
			(field >= fieldACStart && field <= fieldACEnd)
	}

	isACField := func(field int) bool {
		return field >= fieldACStart && field <= fieldACEnd
	}

	isAutocompleteField := func(field int) bool {
		return field == fieldContext || field == fieldIntent ||
			(field >= fieldACStart && field <= fieldACEnd)
	}

	updateAutocomplete := func() {
		if m.fileAutocomplete != nil && isAutocompleteField(m.pendingBallFormField) {
			text := m.textInput.Value()
			cursorPos := m.textInput.Position()
			m.fileAutocomplete.UpdateFromText(text, cursorPos)
		} else if m.fileAutocomplete != nil {
			m.fileAutocomplete.Reset()
		}
	}

	saveCurrentFieldValue := func() {
		switch m.pendingBallFormField {
		case fieldContext:
			m.pendingBallContext = strings.TrimSpace(m.contextInput.Value())
		case fieldIntent:
			m.pendingBallIntent = strings.TrimSpace(m.textInput.Value())
		default:
			value := strings.TrimSpace(m.textInput.Value())
			if m.pendingBallFormField == fieldTags {
				m.pendingBallTags = value
			} else if m.pendingBallFormField == fieldBlockingReason && m.pendingBallBlockingReason == 4 {
				// Custom blocking reason text
				m.pendingBallCustomReason = value
			} else if isACField(m.pendingBallFormField) {
				acIndex := m.pendingBallFormField - fieldACStart
				if acIndex < len(m.pendingAcceptanceCriteria) {
					if value == "" {
						m.pendingAcceptanceCriteria = append(
							m.pendingAcceptanceCriteria[:acIndex],
							m.pendingAcceptanceCriteria[acIndex+1:]...,
						)
					} else {
						m.pendingAcceptanceCriteria[acIndex] = value
					}
				} else {
					// On the "new AC" field - save content for restoration when navigating back
					m.pendingNewAC = value
				}
			}
		}
	}

	recalcFieldIndices := func() (int, int, int, int, int, int, int, int) {
		newFieldACEnd := fieldACStart + len(m.pendingAcceptanceCriteria)
		newFieldTags := newFieldACEnd + 1
		newFieldSession := newFieldTags + 1
		newFieldModelSize := newFieldSession + 1
		newFieldPriority := newFieldModelSize + 1
		newFieldBlockingReason := newFieldPriority + 1
		newFieldDependsOn := newFieldBlockingReason + 1
		newFieldSave := newFieldDependsOn + 1
		return newFieldACEnd, newFieldTags, newFieldSession, newFieldModelSize, newFieldPriority, newFieldBlockingReason, newFieldDependsOn, newFieldSave
	}

	loadFieldValue := func(field int) {
		acEnd, tagsField, _, _, _, blockingReasonField, _, _ := recalcFieldIndices()

		m.textInput.Reset()
		switch field {
		case fieldContext:
			m.contextInput.SetValue(m.pendingBallContext)
			m.contextInput.Focus()
			m.textInput.Blur()
			adjustStandaloneEditContextHeight(&m)
		case fieldIntent:
			m.contextInput.Blur()
			m.textInput.SetValue(m.pendingBallIntent)
			m.textInput.Placeholder = "What is this ball about? (50 char recommended)"
			m.textInput.Focus()
		default:
			m.contextInput.Blur()
			if field == tagsField {
				m.textInput.SetValue(m.pendingBallTags)
				m.textInput.Placeholder = "tag1, tag2, ..."
				m.textInput.Focus()
			} else if field == blockingReasonField && m.pendingBallBlockingReason == 4 {
				// Custom blocking reason - show text input
				m.textInput.SetValue(m.pendingBallCustomReason)
				m.textInput.Placeholder = "Enter custom blocking reason"
				m.textInput.Focus()
			} else if field >= fieldACStart && field <= acEnd {
				acIndex := field - fieldACStart
				if acIndex < len(m.pendingAcceptanceCriteria) {
					m.textInput.SetValue(m.pendingAcceptanceCriteria[acIndex])
					m.textInput.Placeholder = "Edit acceptance criterion"
				} else {
					// Restore preserved new AC content when navigating back
					m.textInput.SetValue(m.pendingNewAC)
					m.textInput.Placeholder = "New acceptance criterion (Enter on empty = save)"
				}
				m.textInput.Focus()
			} else {
				m.textInput.Blur()
			}
		}
	}

	switch msg.String() {
	case "esc":
		m.done = true
		m.result = nil
		return m, tea.Quit

	case "ctrl+enter", "ctrl+s":
		// Save the ball (ctrl+s is more reliable across terminals)
		saveCurrentFieldValue()
		// Title can be empty if context has content (will auto-generate placeholder)
		if m.pendingBallIntent == "" && m.pendingBallContext == "" {
			m.message = "Title is required (or add context to auto-generate)"
			return m, nil
		}
		return m.finalizeBallEdit()

	case "enter":
		if m.pendingBallFormField == fieldContext {
			var cmd tea.Cmd
			m.contextInput, cmd = m.contextInput.Update(msg)
			adjustStandaloneEditContextHeight(&m)
			// Update pendingBallContext live so title placeholder updates as you type
			m.pendingBallContext = m.contextInput.Value()
			return m, cmd
		} else if m.pendingBallFormField == fieldSave {
			// Save button - finalize ball edit
			saveCurrentFieldValue()
			// Title can be empty if context has content (will auto-generate placeholder)
			if m.pendingBallIntent == "" && m.pendingBallContext == "" {
				m.message = "Title is required (or add context to auto-generate)"
				return m, nil
			}
			return m.finalizeBallEdit()
		} else if m.pendingBallFormField == fieldDependsOn {
			return m.openDependencySelector()
		} else if isACField(m.pendingBallFormField) {
			acIndex := m.pendingBallFormField - fieldACStart
			value := strings.TrimSpace(m.textInput.Value())

			if acIndex == len(m.pendingAcceptanceCriteria) {
				if value == "" {
					saveCurrentFieldValue()
					// Title can be empty if context has content (will auto-generate placeholder)
					if m.pendingBallIntent == "" && m.pendingBallContext == "" {
						m.message = "Title is required (or add context to auto-generate)"
						return m, nil
					}
					return m.finalizeBallEdit()
				} else {
					m.pendingAcceptanceCriteria = append(m.pendingAcceptanceCriteria, value)
					m.pendingNewAC = "" // Clear preserved content since it was added
					m.textInput.Reset()
					m.textInput.Placeholder = "New acceptance criterion (Enter on empty = save)"
					m.pendingBallFormField = fieldACStart + len(m.pendingAcceptanceCriteria)
				}
			} else {
				saveCurrentFieldValue()
				m.pendingBallFormField++
				newACEnd, _, _, _, _, _, _, newSave := recalcFieldIndices()
				maxFieldIndex = newSave
				if m.pendingBallFormField > newACEnd {
					_, newFieldTags, _, _, _, _, _, _ := recalcFieldIndices()
					m.pendingBallFormField = newFieldTags
				}
				loadFieldValue(m.pendingBallFormField)
			}
		} else {
			saveCurrentFieldValue()
			m.pendingBallFormField++
			_, _, _, _, _, _, _, newSave := recalcFieldIndices()
			maxFieldIndex = newSave
			if m.pendingBallFormField > maxFieldIndex {
				m.pendingBallFormField = maxFieldIndex
			}
			loadFieldValue(m.pendingBallFormField)
		}
		return m, nil

	case "up":
		if m.fileAutocomplete != nil && m.fileAutocomplete.Active && len(m.fileAutocomplete.Suggestions) > 0 {
			m.fileAutocomplete.SelectPrev()
			return m, nil
		}
		saveCurrentFieldValue()
		m.pendingBallFormField--
		_, _, _, _, _, _, _, newSave := recalcFieldIndices()
		maxFieldIndex = newSave
		if m.pendingBallFormField < 0 {
			m.pendingBallFormField = maxFieldIndex
		}
		loadFieldValue(m.pendingBallFormField)
		return m, nil

	case "down":
		if m.fileAutocomplete != nil && m.fileAutocomplete.Active && len(m.fileAutocomplete.Suggestions) > 0 {
			m.fileAutocomplete.SelectNext()
			return m, nil
		}
		saveCurrentFieldValue()
		m.pendingBallFormField++
		_, _, _, _, _, _, _, newSave := recalcFieldIndices()
		maxFieldIndex = newSave
		if m.pendingBallFormField > maxFieldIndex {
			m.pendingBallFormField = 0
		}
		loadFieldValue(m.pendingBallFormField)
		return m, nil

	case "tab":
		// If autocomplete is active, accept the completion
		if m.fileAutocomplete != nil && m.fileAutocomplete.Active && len(m.fileAutocomplete.Suggestions) > 0 {
			if m.pendingBallFormField == fieldContext {
				newText := m.fileAutocomplete.ApplyCompletion(m.contextInput.Value())
				m.contextInput.SetValue(newText)
				adjustStandaloneEditContextHeight(&m)
			} else {
				text := m.textInput.Value()
				newText := m.fileAutocomplete.ApplyCompletion(text)
				m.textInput.SetValue(newText)
				m.textInput.SetCursor(len(newText))
			}
			m.fileAutocomplete.Reset()
			return m, nil
		}

		// Tab always moves to next field
		// For selection fields, also toggle to next option before moving
		_, _, sessionField, modelSizeField, priorityField, blockingReasonField, _, _ := recalcFieldIndices()
		if m.pendingBallFormField == sessionField {
			// Toggle to next session option
			m.pendingBallSession++
			if m.pendingBallSession >= numSessionOptions {
				m.pendingBallSession = 0
			}
		} else if m.pendingBallFormField == modelSizeField {
			// Toggle to next model size option
			m.pendingBallModelSize++
			if m.pendingBallModelSize >= numModelSizeOptions {
				m.pendingBallModelSize = 0
			}
		} else if m.pendingBallFormField == priorityField {
			// Toggle to next priority option
			m.pendingBallPriority++
			if m.pendingBallPriority >= numPriorityOptions {
				m.pendingBallPriority = 0
			}
		} else if m.pendingBallFormField == blockingReasonField {
			// Toggle to next blocking reason option
			m.pendingBallBlockingReason++
			if m.pendingBallBlockingReason >= numBlockingReasonOptions {
				m.pendingBallBlockingReason = 0
			}
		} else {
			// For text fields, save current value
			saveCurrentFieldValue()
		}
		// Move to next field
		newACEnd, newFieldTags, _, _, _, _, _, newSave := recalcFieldIndices()
		if m.pendingBallFormField == newACEnd {
			m.pendingBallFormField = newFieldTags
		} else {
			m.pendingBallFormField++
			maxFieldIndex = newSave
			if m.pendingBallFormField > maxFieldIndex {
				m.pendingBallFormField = 0
			}
		}
		loadFieldValue(m.pendingBallFormField)
		return m, nil

	case " ":
		if m.fileAutocomplete != nil && m.fileAutocomplete.Active && len(m.fileAutocomplete.Suggestions) > 0 {
			m.fileAutocomplete.Reset()
		}
		// Fall through to handle space in text input

	case "left", "right":
		if m.pendingBallFormField == fieldSession {
			if msg.String() == "left" && m.pendingBallSession > 0 {
				m.pendingBallSession--
			} else if msg.String() == "right" && m.pendingBallSession < numSessionOptions-1 {
				m.pendingBallSession++
			}
			return m, nil
		} else if m.pendingBallFormField == fieldModelSize {
			if msg.String() == "left" && m.pendingBallModelSize > 0 {
				m.pendingBallModelSize--
			} else if msg.String() == "right" && m.pendingBallModelSize < numModelSizeOptions-1 {
				m.pendingBallModelSize++
			}
			return m, nil
		} else if m.pendingBallFormField == fieldPriority {
			if msg.String() == "left" && m.pendingBallPriority > 0 {
				m.pendingBallPriority--
			} else if msg.String() == "right" && m.pendingBallPriority < numPriorityOptions-1 {
				m.pendingBallPriority++
			}
			return m, nil
		} else if m.pendingBallFormField == fieldBlockingReason {
			if msg.String() == "left" && m.pendingBallBlockingReason > 0 {
				m.pendingBallBlockingReason--
			} else if msg.String() == "right" && m.pendingBallBlockingReason < numBlockingReasonOptions-1 {
				m.pendingBallBlockingReason++
			}
			// Load text input if switching to/from custom mode
			loadFieldValue(m.pendingBallFormField)
			return m, nil
		}
	}

	// Handle text input for text fields
	if isTextInputField(m.pendingBallFormField) {
		var cmd tea.Cmd
		if m.pendingBallFormField == fieldContext {
			m.contextInput, cmd = m.contextInput.Update(msg)
			adjustStandaloneEditContextHeight(&m)
			// Update pendingBallContext live so title placeholder updates as you type
			m.pendingBallContext = m.contextInput.Value()
		} else {
			m.textInput, cmd = m.textInput.Update(msg)
			updateAutocomplete()
		}
		return m, cmd
	}

	return m, nil
}

func (m StandaloneEditModel) openDependencySelector() (tea.Model, tea.Cmd) {
	balls, err := m.store.LoadBalls()
	if err != nil {
		m.message = "Error loading balls: " + err.Error()
		return m, nil
	}

	var selectableBalls []*session.Ball
	for _, ball := range balls {
		// Exclude the ball being edited and complete balls
		if ball.ID != m.ball.ID && ball.State != session.StateComplete && ball.State != session.StateResearched {
			selectableBalls = append(selectableBalls, ball)
		}
	}

	if len(selectableBalls) == 0 {
		m.message = "No non-complete balls to select as dependencies"
		return m, nil
	}

	m.dependencySelectBalls = selectableBalls
	m.dependencySelectIndex = 0
	m.dependencySelectActive = make(map[string]bool)
	for _, depID := range m.pendingBallDependsOn {
		m.dependencySelectActive[depID] = true
	}
	m.inDependencySelector = true

	return m, nil
}

func (m StandaloneEditModel) handleDependencySelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.inDependencySelector = false
		m.dependencySelectBalls = nil
		m.dependencySelectActive = nil
		m.message = "Cancelled"
		return m, nil

	case "up", "k":
		if m.dependencySelectIndex > 0 {
			m.dependencySelectIndex--
		}
		return m, nil

	case "down", "j":
		if m.dependencySelectIndex < len(m.dependencySelectBalls)-1 {
			m.dependencySelectIndex++
		}
		return m, nil

	case " ":
		if m.dependencySelectIndex < len(m.dependencySelectBalls) {
			ball := m.dependencySelectBalls[m.dependencySelectIndex]
			m.dependencySelectActive[ball.ID] = !m.dependencySelectActive[ball.ID]
		}
		return m, nil

	case "enter":
		var selected []string
		for id, isSelected := range m.dependencySelectActive {
			if isSelected {
				selected = append(selected, id)
			}
		}
		sort.Strings(selected)
		m.pendingBallDependsOn = selected

		m.inDependencySelector = false
		m.dependencySelectBalls = nil
		m.dependencySelectActive = nil
		if len(m.pendingBallDependsOn) > 0 {
			m.message = fmt.Sprintf("Selected %d dependencies", len(m.pendingBallDependsOn))
		} else {
			m.message = "Cleared dependencies"
		}
		return m, nil
	}
	return m, nil
}

func (m StandaloneEditModel) finalizeBallEdit() (tea.Model, tea.Cmd) {
	// Include any preserved new AC content that wasn't added via Enter
	if m.pendingNewAC != "" {
		m.pendingAcceptanceCriteria = append(m.pendingAcceptanceCriteria, m.pendingNewAC)
		m.pendingNewAC = ""
	}

	priorities := []session.Priority{session.PriorityLow, session.PriorityMedium, session.PriorityHigh, session.PriorityUrgent}
	priority := priorities[m.pendingBallPriority]

	modelSizes := []session.ModelSize{session.ModelSizeBlank, session.ModelSizeSmall, session.ModelSizeMedium, session.ModelSizeLarge}
	modelSize := modelSizes[m.pendingBallModelSize]

	var tags []string
	if m.pendingBallTags != "" {
		tagList := strings.Split(m.pendingBallTags, ",")
		for _, tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	if m.pendingBallSession > 0 {
		realSessions := []*session.JuggleSession{}
		for _, sess := range m.sessions {
			if sess.ID != PseudoSessionAll && sess.ID != PseudoSessionUntagged {
				realSessions = append(realSessions, sess)
			}
		}
		if m.pendingBallSession-1 < len(realSessions) {
			tags = append(tags, realSessions[m.pendingBallSession-1].ID)
		}
	}

	// Handle blocking reason
	// Options: 0=blank, 1=Human needed, 2=Waiting for dependency, 3=Needs research, 4=custom
	var blockedReason string
	if m.pendingBallBlockingReason == 1 {
		// "Human needed" auto-adds the human-needed tag
		blockedReason = "Human needed"
		// Add human-needed tag if not already present
		hasHumanNeededTag := false
		for _, tag := range tags {
			if tag == "human-needed" {
				hasHumanNeededTag = true
				break
			}
		}
		if !hasHumanNeededTag {
			tags = append(tags, "human-needed")
		}
	} else if m.pendingBallBlockingReason == 2 {
		blockedReason = "Waiting for dependency"
	} else if m.pendingBallBlockingReason == 3 {
		blockedReason = "Needs research"
	} else if m.pendingBallBlockingReason == 4 {
		blockedReason = m.pendingBallCustomReason
	}

	// Update the ball with new values
	m.ball.Title = m.pendingBallIntent
	m.ball.Context = m.pendingBallContext
	m.ball.Priority = priority
	m.ball.Tags = tags
	m.ball.ModelSize = modelSize
	m.ball.BlockedReason = blockedReason

	if len(m.pendingAcceptanceCriteria) > 0 {
		m.ball.SetAcceptanceCriteria(m.pendingAcceptanceCriteria)
	} else {
		m.ball.AcceptanceCriteria = nil
	}

	if len(m.pendingBallDependsOn) > 0 {
		m.ball.SetDependencies(m.pendingBallDependsOn)
	} else {
		m.ball.DependsOn = nil
	}

	m.ball.UpdateActivity()

	err := m.store.UpdateBall(m.ball)
	if err != nil {
		m.err = err
		m.done = true
		return m, tea.Quit
	}

	m.result = m.ball
	m.done = true
	return m, tea.Quit
}

func (m StandaloneEditModel) View() string {
	if m.inDependencySelector {
		return m.renderDependencySelector()
	}
	return m.renderForm()
}

func (m StandaloneEditModel) renderForm() string {
	var b strings.Builder

	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render(fmt.Sprintf("Edit Ball: %s", m.ball.ID))
	b.WriteString(titleStyled + "\n\n")

	const (
		fieldContext = 0
		fieldIntent  = 1
		fieldACStart = 2
	)
	fieldACEnd := fieldACStart + len(m.pendingAcceptanceCriteria)
	fieldTags := fieldACEnd + 1
	fieldSession := fieldTags + 1
	fieldModelSize := fieldSession + 1
	fieldPriority := fieldModelSize + 1
	fieldBlockingReason := fieldPriority + 1
	fieldDependsOn := fieldBlockingReason + 1
	fieldSave := fieldDependsOn + 1

	sessionOptions := []string{"(none)"}
	for _, sess := range m.sessions {
		if sess.ID != PseudoSessionAll && sess.ID != PseudoSessionUntagged {
			sessionOptions = append(sessionOptions, sess.ID)
		}
	}

	activeFieldStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	normalStyle := lipgloss.NewStyle()
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	optionSelectedStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0"))
	optionNormalStyle := lipgloss.NewStyle().Faint(true)
	acNumberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	editingACStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("240"))
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))

	// Context field
	labelStyle := normalStyle
	if m.pendingBallFormField == fieldContext {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Context:"))
	if m.pendingBallFormField == fieldContext {
		b.WriteString("\n")
		b.WriteString(m.contextInput.View())
		if popup := m.renderAutocompletePopup(); popup != "" {
			b.WriteString(popup)
		}
	} else {
		b.WriteString(" ")
		if m.pendingBallContext == "" {
			b.WriteString(optionNormalStyle.Render("(empty)"))
		} else {
			wrapped := wrapText(m.pendingBallContext, 60)
			lines := strings.Split(wrapped, "\n")
			if len(lines) == 1 {
				b.WriteString(m.pendingBallContext)
			} else {
				b.WriteString(lines[0])
				indent := "         "
				for i := 1; i < len(lines); i++ {
					b.WriteString("\n" + indent + lines[i])
				}
			}
		}
	}
	b.WriteString("\n")

	// Title field
	labelStyle = normalStyle
	if m.pendingBallFormField == fieldIntent {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Title: "))
	if m.pendingBallFormField == fieldIntent {
		b.WriteString(m.textInput.View())
		titleLen := len(m.textInput.Value())
		countStyle := lipgloss.NewStyle().Faint(true)
		if titleLen >= 50 {
			countStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		} else if titleLen >= 40 {
			countStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
		}
		b.WriteString(countStyle.Render(fmt.Sprintf(" (%d/50)", titleLen)))
		if popup := m.renderAutocompletePopup(); popup != "" {
			b.WriteString(popup)
		}
	} else {
		if m.pendingBallIntent == "" {
			// Show context-derived placeholder when unfocused, or (empty) if no context
			if m.pendingBallContext != "" {
				placeholder := generateTitlePlaceholderFromContext(m.pendingBallContext)
				if placeholder != "" {
					b.WriteString(optionNormalStyle.Render(placeholder))
				} else {
					b.WriteString(optionNormalStyle.Render("(empty)"))
				}
			} else {
				b.WriteString(optionNormalStyle.Render("(empty)"))
			}
		} else {
			titleLen := len(m.pendingBallIntent)
			b.WriteString(m.pendingBallIntent)
			if titleLen >= 50 {
				countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
				b.WriteString(countStyle.Render(fmt.Sprintf(" (%d/50)", titleLen)))
			} else if titleLen >= 40 {
				countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
				b.WriteString(countStyle.Render(fmt.Sprintf(" (%d/50)", titleLen)))
			}
		}
	}
	b.WriteString("\n\n")

	// Acceptance Criteria
	acLabel := normalStyle
	isOnACField := m.pendingBallFormField >= fieldACStart && m.pendingBallFormField <= fieldACEnd
	if isOnACField {
		acLabel = activeFieldStyle
	}
	acHeaderText := "Acceptance Criteria:"
	if len(m.pendingAcceptanceCriteria) == 0 && !isOnACField {
		acHeaderText += warningStyle.Render(" (none - consider adding criteria)")
	}
	b.WriteString(acLabel.Render(acHeaderText) + "\n")

	for i, ac := range m.pendingAcceptanceCriteria {
		acFieldIndex := fieldACStart + i
		if m.pendingBallFormField == acFieldIndex {
			b.WriteString(acNumberStyle.Render(fmt.Sprintf("  %d. ", i+1)))
			b.WriteString(m.textInput.View())
			if popup := m.renderAutocompletePopup(); popup != "" {
				b.WriteString(popup)
			}
		} else {
			b.WriteString(acNumberStyle.Render(fmt.Sprintf("  %d. ", i+1)))
			b.WriteString(ac)
		}
		b.WriteString("\n")
	}

	if m.pendingBallFormField == fieldACEnd {
		b.WriteString(editingACStyle.Render("  + "))
		b.WriteString(m.textInput.View())
		if popup := m.renderAutocompletePopup(); popup != "" {
			b.WriteString(popup)
		}
		b.WriteString("\n")
	} else {
		// Show pending new AC content if exists, otherwise show placeholder
		if m.pendingNewAC != "" {
			b.WriteString(acNumberStyle.Render("  + "))
			b.WriteString(m.pendingNewAC)
		} else {
			b.WriteString(optionNormalStyle.Render("  + (add criterion)"))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Tags field
	labelStyle = normalStyle
	if m.pendingBallFormField == fieldTags {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Tags: "))
	if m.pendingBallFormField == fieldTags {
		b.WriteString(m.textInput.View())
	} else {
		if m.pendingBallTags == "" {
			b.WriteString(optionNormalStyle.Render("(none)"))
		} else {
			b.WriteString(m.pendingBallTags)
		}
	}
	b.WriteString("\n")

	// Session field
	labelStyle = normalStyle
	if m.pendingBallFormField == fieldSession {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Session: "))
	for i, opt := range sessionOptions {
		if i > 0 {
			b.WriteString(" ")
		}
		if i == m.pendingBallSession {
			if m.pendingBallFormField == fieldSession {
				b.WriteString(optionSelectedStyle.Render(" " + opt + " "))
			} else {
				b.WriteString(lipgloss.NewStyle().Bold(true).Render(opt))
			}
		} else {
			b.WriteString(optionNormalStyle.Render(opt))
		}
	}
	b.WriteString("\n")

	// Model Size field
	modelSizeOptions := []string{"(default)", "small", "medium", "large"}
	labelStyle = normalStyle
	if m.pendingBallFormField == fieldModelSize {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Model Size: "))
	for i, opt := range modelSizeOptions {
		if i > 0 {
			b.WriteString(" ")
		}
		if i == m.pendingBallModelSize {
			if m.pendingBallFormField == fieldModelSize {
				b.WriteString(optionSelectedStyle.Render(" " + opt + " "))
			} else {
				b.WriteString(lipgloss.NewStyle().Bold(true).Render(opt))
			}
		} else {
			b.WriteString(optionNormalStyle.Render(opt))
		}
	}
	b.WriteString("\n")

	// Priority field
	priorityOptions := []string{"low", "medium", "high", "urgent"}
	priorityColors := []string{"245", "6", "214", "196"} // gray, cyan, orange, red
	labelStyle = normalStyle
	if m.pendingBallFormField == fieldPriority {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Priority: "))
	for i, opt := range priorityOptions {
		if i > 0 {
			b.WriteString(" | ")
		}
		if i == m.pendingBallPriority {
			if m.pendingBallFormField == fieldPriority {
				b.WriteString(optionSelectedStyle.Render(opt))
			} else {
				// Use priority color for selected option when not focused
				colorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(priorityColors[i]))
				b.WriteString(colorStyle.Render(opt))
			}
		} else {
			b.WriteString(optionNormalStyle.Render(opt))
		}
	}
	b.WriteString("\n")

	// Blocking Reason field
	blockingReasonOptions := []string{"(blank)", "Human needed", "Waiting for dependency", "Needs research", "(custom)"}
	labelStyle = normalStyle
	if m.pendingBallFormField == fieldBlockingReason {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Blocking Reason: "))

	// Check if we're on the custom option AND focused - show text input
	if m.pendingBallFormField == fieldBlockingReason && m.pendingBallBlockingReason == 4 {
		// Show custom text input
		b.WriteString(optionSelectedStyle.Render("(custom): "))
		b.WriteString(m.textInput.View())
	} else {
		for i, opt := range blockingReasonOptions {
			if i > 0 {
				b.WriteString(" | ")
			}
			if i == m.pendingBallBlockingReason {
				if m.pendingBallFormField == fieldBlockingReason {
					b.WriteString(optionSelectedStyle.Render(opt))
				} else {
					// Show selected option with appropriate color when not focused
					if i == 0 {
						b.WriteString(optionNormalStyle.Render(opt))
					} else if i == 4 && m.pendingBallCustomReason != "" {
						// Show custom reason text when not focused
						customStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
						b.WriteString(customStyle.Render(m.pendingBallCustomReason))
					} else {
						reasonStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
						b.WriteString(reasonStyle.Render(opt))
					}
				}
			} else {
				b.WriteString(optionNormalStyle.Render(opt))
			}
		}
	}
	b.WriteString("\n")

	// Depends On field
	labelStyle = normalStyle
	if m.pendingBallFormField == fieldDependsOn {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Depends On: "))
	if len(m.pendingBallDependsOn) == 0 {
		if m.pendingBallFormField == fieldDependsOn {
			b.WriteString(optionSelectedStyle.Render("(none) - press Enter to select"))
		} else {
			b.WriteString(optionNormalStyle.Render("(none)"))
		}
	} else {
		depDisplay := strings.Join(m.pendingBallDependsOn, ", ")
		if m.pendingBallFormField == fieldDependsOn {
			b.WriteString(selectedStyle.Render(depDisplay) + optionNormalStyle.Render(" - press Enter to edit"))
		} else {
			b.WriteString(depDisplay)
		}
	}
	b.WriteString("\n\n")

	// Save button
	saveButtonStyle := lipgloss.NewStyle().Padding(0, 2)
	if m.pendingBallFormField == fieldSave {
		saveButtonStyle = saveButtonStyle.Bold(true).Background(lipgloss.Color("2")).Foreground(lipgloss.Color("0"))
	} else {
		saveButtonStyle = saveButtonStyle.Foreground(lipgloss.Color("2")).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("2"))
	}
	b.WriteString(saveButtonStyle.Render("[ Save ]") + "\n\n")

	// Message
	if m.message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
		b.WriteString(msgStyle.Render(m.message) + "\n\n")
	}

	// Help text
	helpStyle := lipgloss.NewStyle().Faint(true)
	b.WriteString(helpStyle.Render("↑/↓ = navigate | Tab = next | ←/→ = cycle options | Enter = next/add | Ctrl+S = save | Esc = cancel"))

	return b.String()
}

func (m StandaloneEditModel) renderDependencySelector() string {
	var b strings.Builder

	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render("Select Dependencies")
	b.WriteString(titleStyled + "\n\n")

	helpStyle := lipgloss.NewStyle().Faint(true)
	b.WriteString(helpStyle.Render("Space: toggle • Enter: confirm • Esc: cancel") + "\n\n")

	selectedStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0"))
	normalStyle := lipgloss.NewStyle()
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	for i, ball := range m.dependencySelectBalls {
		isSelected := m.dependencySelectActive[ball.ID]
		isCursor := i == m.dependencySelectIndex

		check := "[ ]"
		if isSelected {
			check = checkStyle.Render("[✓]")
		}

		line := fmt.Sprintf("%s %s: %s", check, ball.ID, ball.Title)
		if len(line) > 70 {
			line = line[:67] + "..."
		}

		if isCursor {
			b.WriteString(selectedStyle.Render(" " + line + " "))
		} else {
			b.WriteString(normalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m StandaloneEditModel) renderAutocompletePopup() string {
	if m.fileAutocomplete == nil || !m.fileAutocomplete.Active || len(m.fileAutocomplete.Suggestions) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")

	selectedStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0"))
	normalStyle := lipgloss.NewStyle().Faint(true)

	maxShow := 5
	if len(m.fileAutocomplete.Suggestions) < maxShow {
		maxShow = len(m.fileAutocomplete.Suggestions)
	}

	for i := 0; i < maxShow; i++ {
		suggestion := m.fileAutocomplete.Suggestions[i]
		if i == m.fileAutocomplete.Selected {
			b.WriteString(selectedStyle.Render("  " + suggestion))
		} else {
			b.WriteString(normalStyle.Render("  " + suggestion))
		}
		b.WriteString("\n")
	}

	if len(m.fileAutocomplete.Suggestions) > maxShow {
		b.WriteString(normalStyle.Render(fmt.Sprintf("  ... and %d more", len(m.fileAutocomplete.Suggestions)-maxShow)))
		b.WriteString("\n")
	}

	return b.String()
}

// Result returns the result of the ball editing
func (m StandaloneEditModel) Result() StandaloneEditResult {
	return StandaloneEditResult{
		Ball:      m.result,
		Cancelled: m.result == nil && m.err == nil,
		Err:       m.err,
	}
}

// Done returns whether the model is done
func (m StandaloneEditModel) Done() bool {
	return m.done
}

// adjustStandaloneEditContextHeight adjusts the context textarea height based on content
func adjustStandaloneEditContextHeight(m *StandaloneEditModel) {
	content := m.contextInput.Value()
	if content == "" {
		m.contextInput.SetHeight(1)
		return
	}

	wrappedLines := 0
	for _, line := range strings.Split(content, "\n") {
		if len(line) > 58 {
			wrappedLines += (len(line) / 58) + 1
		} else {
			wrappedLines++
		}
	}

	height := wrappedLines + 1 // +1 for cursor line
	if height < 1 {
		height = 1
	}
	if height > 10 {
		height = 10
	}
	m.contextInput.SetHeight(height)
}
