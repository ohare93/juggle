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

// StandaloneBallModel is a TUI model for standalone ball creation (from CLI plan command)
// It exits after creating or cancelling
type StandaloneBallModel struct {
	store        *session.Store
	sessionStore *session.SessionStore
	sessions     []*session.JuggleSession

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
	done    bool     // True when form is completed (either save or cancel)
	result  *session.Ball // The created ball (nil if cancelled)
	err     error
}

// StandaloneBallResult contains the result of the standalone ball creation
type StandaloneBallResult struct {
	Ball      *session.Ball // Created ball, nil if cancelled
	Cancelled bool          // True if user cancelled
	Err       error         // Error if any
}

// NewStandaloneBallModel creates a new standalone ball creation model
func NewStandaloneBallModel(store *session.Store, sessionStore *session.SessionStore) StandaloneBallModel {
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

	return StandaloneBallModel{
		store:               store,
		sessionStore:        sessionStore,
		textInput:           ti,
		contextInput:        ta,
		pendingBallPriority: 1, // Default to medium
		fileAutocomplete:    NewAutocompleteState(store.ProjectDir()),
	}
}

// PrePopulate sets initial values for the form fields from flags
func (m *StandaloneBallModel) PrePopulate(intent, context string, tags []string, sessionID string, priority string, modelSize string, acceptanceCriteria []string, dependsOn []string) {
	m.pendingBallIntent = intent
	m.pendingBallContext = context
	m.contextInput.SetValue(context)

	if len(tags) > 0 {
		m.pendingBallTags = strings.Join(tags, ", ")
	}

	// Map priority to index
	switch priority {
	case "low":
		m.pendingBallPriority = 0
	case "medium", "":
		m.pendingBallPriority = 1
	case "high":
		m.pendingBallPriority = 2
	case "urgent":
		m.pendingBallPriority = 3
	}

	// Map model size to index
	switch modelSize {
	case "", "default":
		m.pendingBallModelSize = 0
	case "small":
		m.pendingBallModelSize = 1
	case "medium":
		m.pendingBallModelSize = 2
	case "large":
		m.pendingBallModelSize = 3
	}

	// Set acceptance criteria
	m.pendingAcceptanceCriteria = acceptanceCriteria

	// Store session ID to match later when sessions are loaded
	if sessionID != "" {
		// We'll match this when sessions are loaded
		m.pendingBallTags = sessionID // Temporarily store - will fix when sessions load
	}

	m.pendingBallDependsOn = dependsOn
	adjustStandaloneContextHeight(m)
}

func (m StandaloneBallModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		loadSessionsForStandalone(m.sessionStore),
	)
}

type loadedSessionsForStandaloneMsg struct {
	sessions []*session.JuggleSession
	err      error
}

func loadSessionsForStandalone(sessionStore *session.SessionStore) tea.Cmd {
	return func() tea.Msg {
		if sessionStore == nil {
			return loadedSessionsForStandaloneMsg{sessions: nil}
		}
		sessions, err := sessionStore.ListSessions()
		if err != nil {
			return loadedSessionsForStandaloneMsg{err: err}
		}
		return loadedSessionsForStandaloneMsg{sessions: sessions}
	}
}

func (m StandaloneBallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedSessionsForStandaloneMsg:
		if msg.err != nil {
			m.message = "Warning: couldn't load sessions: " + msg.err.Error()
		} else {
			m.sessions = msg.sessions
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

func (m StandaloneBallModel) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			adjustStandaloneContextHeight(&m)
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
		// Create the ball (ctrl+s is more reliable across terminals)
		saveCurrentFieldValue()
		if m.pendingBallIntent == "" {
			m.message = "Title is required"
			return m, nil
		}
		return m.finalizeBallCreation()

	case "enter":
		if m.pendingBallFormField == fieldContext {
			var cmd tea.Cmd
			m.contextInput, cmd = m.contextInput.Update(msg)
			adjustStandaloneContextHeight(&m)
			// Update pendingBallContext live so title placeholder updates as you type
			m.pendingBallContext = m.contextInput.Value()
			return m, cmd
		} else if m.pendingBallFormField == fieldSave {
			// Save button - finalize ball creation
			saveCurrentFieldValue()
			if m.pendingBallIntent == "" {
				m.message = "Title is required"
				return m, nil
			}
			return m.finalizeBallCreation()
		} else if m.pendingBallFormField == fieldDependsOn {
			return m.openDependencySelector()
		} else if isACField(m.pendingBallFormField) {
			acIndex := m.pendingBallFormField - fieldACStart
			value := strings.TrimSpace(m.textInput.Value())

			if acIndex == len(m.pendingAcceptanceCriteria) {
				if value == "" {
					saveCurrentFieldValue()
					if m.pendingBallIntent == "" {
						m.message = "Title is required"
						return m, nil
					}
					return m.finalizeBallCreation()
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
				adjustStandaloneContextHeight(&m)
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
			adjustStandaloneContextHeight(&m)
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

func (m StandaloneBallModel) openDependencySelector() (tea.Model, tea.Cmd) {
	balls, err := m.store.LoadBalls()
	if err != nil {
		m.message = "Error loading balls: " + err.Error()
		return m, nil
	}

	var selectableBalls []*session.Ball
	for _, ball := range balls {
		if ball.State != session.StateComplete && ball.State != session.StateResearched {
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

func (m StandaloneBallModel) handleDependencySelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m StandaloneBallModel) finalizeBallCreation() (tea.Model, tea.Cmd) {
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

	ball, err := session.NewBall(m.store.ProjectDir(), m.pendingBallIntent, priority)
	if err != nil {
		m.err = err
		m.done = true
		return m, tea.Quit
	}

	ball.State = session.StatePending
	ball.Context = m.pendingBallContext
	ball.Tags = tags
	ball.ModelSize = modelSize
	ball.BlockedReason = blockedReason

	if len(m.pendingAcceptanceCriteria) > 0 {
		ball.SetAcceptanceCriteria(m.pendingAcceptanceCriteria)
	}

	if len(m.pendingBallDependsOn) > 0 {
		ball.SetDependencies(m.pendingBallDependsOn)
	}

	err = m.store.AppendBall(ball)
	if err != nil {
		m.err = err
		m.done = true
		return m, tea.Quit
	}

	m.result = ball
	m.done = true
	return m, tea.Quit
}

func (m StandaloneBallModel) View() string {
	if m.inDependencySelector {
		return m.renderDependencySelector()
	}
	return m.renderForm()
}

func (m StandaloneBallModel) renderForm() string {
	var b strings.Builder

	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render("Create New Ball")
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

func (m StandaloneBallModel) renderDependencySelector() string {
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

func (m StandaloneBallModel) renderAutocompletePopup() string {
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

// Result returns the result of the ball creation
func (m StandaloneBallModel) Result() StandaloneBallResult {
	return StandaloneBallResult{
		Ball:      m.result,
		Cancelled: m.result == nil && m.err == nil,
		Err:       m.err,
	}
}

// Done returns whether the model is done
func (m StandaloneBallModel) Done() bool {
	return m.done
}

// adjustStandaloneContextHeight adjusts the context textarea height based on content
func adjustStandaloneContextHeight(m *StandaloneBallModel) {
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
