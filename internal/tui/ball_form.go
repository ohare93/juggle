package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
)

// finalizeBallCreation creates the ball with the collected intent and acceptance criteria
func (m Model) finalizeBallCreation() (tea.Model, tea.Cmd) {
	// Include any preserved new AC content that wasn't added via Enter
	if m.pendingNewAC != "" {
		m.pendingAcceptanceCriteria = append(m.pendingAcceptanceCriteria, m.pendingNewAC)
		m.pendingNewAC = ""
	}

	// Auto-generate title from context if title is empty but context has content
	if m.pendingBallIntent == "" && m.pendingBallContext != "" {
		m.pendingBallIntent = generateTitlePlaceholderFromContext(m.pendingBallContext)
	}

	// Map priority index to Priority constant
	priorities := []session.Priority{session.PriorityLow, session.PriorityMedium, session.PriorityHigh, session.PriorityUrgent}
	priority := priorities[m.pendingBallPriority]

	// Map model size index to ModelSize constant
	modelSizes := []session.ModelSize{session.ModelSizeBlank, session.ModelSizeSmall, session.ModelSizeMedium, session.ModelSizeLarge}
	modelSize := modelSizes[m.pendingBallModelSize]

	// Build tags list
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

	// Add session tag if selected in form (0 = none, 1+ = session index)
	if m.pendingBallSession > 0 {
		// Get real sessions (excluding pseudo-sessions)
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

	// Check if we're editing an existing ball or creating a new one
	if m.inputAction == actionEdit && m.editingBall != nil {
		// Update existing ball
		ball := m.editingBall
		ball.Context = m.pendingBallContext
		ball.Title = m.pendingBallIntent
		ball.Priority = priority
		ball.Tags = tags
		ball.ModelSize = modelSize
		ball.BlockedReason = blockedReason

		// Update state based on blocking reason changes:
		// - If blocking reason is set and ball is not blocked -> set to blocked
		// - If blocking reason is cleared and ball is blocked -> set to in_progress
		if blockedReason != "" && ball.State != session.StateBlocked {
			ball.State = session.StateBlocked
		} else if blockedReason == "" && ball.State == session.StateBlocked {
			// Unblock: revert to in_progress (valid transition: blocked -> in_progress)
			ball.State = session.StateInProgress
		}

		// Set acceptance criteria
		if len(m.pendingAcceptanceCriteria) > 0 {
			ball.SetAcceptanceCriteria(m.pendingAcceptanceCriteria)
		} else {
			ball.AcceptanceCriteria = nil
		}

		// Set dependencies
		if len(m.pendingBallDependsOn) > 0 {
			ball.SetDependencies(m.pendingBallDependsOn)
		} else {
			ball.DependsOn = nil
		}

		// Update the ball in store
		err := m.store.UpdateBall(ball)
		if err != nil {
			m.message = "Error updating ball: " + err.Error()
			m.clearPendingBallState()
			m.mode = splitView
			return m, nil
		}

		m.addActivity("Updated ball: " + ball.ID)
		m.message = "Updated ball: " + ball.ID

		// Clear editing state
		m.editingBall = nil
	} else {
		// Create new ball using the store's project directory
		ball, err := session.NewBall(m.store.ProjectDir(), m.pendingBallIntent, priority)
		if err != nil {
			m.message = "Error creating ball: " + err.Error()
			m.clearPendingBallState()
			m.mode = splitView
			return m, nil
		}

		// New balls always start in pending state
		ball.State = session.StatePending
		ball.Context = m.pendingBallContext // Set context from form
		ball.Tags = tags
		ball.ModelSize = modelSize
		ball.BlockedReason = blockedReason

		// Set acceptance criteria if any were collected
		if len(m.pendingAcceptanceCriteria) > 0 {
			ball.SetAcceptanceCriteria(m.pendingAcceptanceCriteria)
		}

		// Set dependencies if any were selected
		if len(m.pendingBallDependsOn) > 0 {
			ball.SetDependencies(m.pendingBallDependsOn)
		}

		// Use the store's working directory
		err = m.store.AppendBall(ball)
		if err != nil {
			m.message = "Error creating ball: " + err.Error()
			m.clearPendingBallState()
			m.mode = splitView
			return m, nil
		}

		m.addActivity("Created ball: " + ball.ID)
		m.message = "Created ball: " + ball.ID
	}

	// Clear pending state
	m.clearPendingBallState()
	m.textInput.Blur()
	m.mode = splitView

	return m, loadBalls(m.store, m.config, m.localOnly)
}

// clearPendingBallState clears all pending ball creation/editing state
func (m *Model) clearPendingBallState() {
	m.pendingBallContext = ""
	m.pendingBallIntent = ""
	m.pendingAcceptanceCriteria = nil
	m.pendingNewAC = ""
	m.pendingBallPriority = 1  // Reset to default (medium)
	m.pendingBallModelSize = 0 // Reset to default
	m.pendingBallTags = ""
	m.pendingBallSession = 0
	m.pendingBallDependsOn = nil
	m.pendingBallBlockingReason = 0 // Reset to blank
	m.pendingBallCustomReason = ""
	m.pendingBallFormField = 0
	m.pendingACEditIndex = -1
	m.dependencySelectBalls = nil
	m.dependencySelectIndex = 0
	m.dependencySelectActive = nil
	m.editingBall = nil
	m.inputAction = actionAdd
	// Clear AC template state
	m.acTemplates = nil
	m.acTemplateSelected = nil
	m.acTemplateCursor = -1
	m.repoLevelACs = nil
	m.sessionLevelACs = nil
}

// generateTitlePlaceholderFromContext generates a title placeholder from context content.
// Returns the first 50 characters trimmed at a word boundary, or empty string if no context.
func generateTitlePlaceholderFromContext(context string) string {
	context = strings.TrimSpace(context)
	if context == "" {
		return ""
	}

	// If context is short enough, use it all
	if len(context) <= 50 {
		return context
	}

	// Find a good word boundary before 50 chars
	// Start from position 50 and look backwards for a space
	cutoff := 50
	for i := cutoff; i > 0; i-- {
		if context[i] == ' ' {
			return strings.TrimSpace(context[:i])
		}
	}

	// No space found, just truncate at 50
	return context[:50]
}

// adjustContextTextareaHeight dynamically adjusts the context textarea height based on content
// The textarea grows as the user types more content, and shrinks when content is deleted
func adjustContextTextareaHeight(m *Model) {
	content := m.contextInput.Value()
	if content == "" {
		m.contextInput.SetHeight(1)
		return
	}

	// Use effective wrap width (textarea has internal padding that reduces usable width)
	// The textarea is set to 60 chars but actual content wraps around 58
	const wrapWidth = 58

	// Count wrapped lines
	wrappedLines := 0
	for _, line := range strings.Split(content, "\n") {
		if len(line) == 0 {
			wrappedLines++
		} else if len(line) > wrapWidth {
			// Long line wraps to multiple display lines
			wrappedLines += (len(line) / wrapWidth) + 1
		} else {
			wrappedLines++
		}
	}

	// Minimum 1 line, maximum 10 lines
	if wrappedLines < 1 {
		wrappedLines = 1
	}
	if wrappedLines > 10 {
		wrappedLines = 10
	}

	m.contextInput.SetHeight(wrappedLines)
}

// handleUnifiedBallFormKey handles keyboard input for the unified ball creation form
// Field order: Context, Title, Acceptance Criteria, Tags, Session, Model Size, Priority, Blocking Reason, Depends On, Save
func (m Model) handleUnifiedBallFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Field indices are dynamic due to variable AC count
	// Order: Context(0), Title(1), ACs(2 to 2+len(ACs)), Tags, Session, ModelSize, Priority, BlockingReason, DependsOn, Save
	const (
		fieldContext = 0
		fieldIntent  = 1 // Title field (was intent)
		fieldACStart = 2 // ACs start at index 2
	)
	// Dynamic field indices calculated after ACs
	fieldACEnd := fieldACStart + len(m.pendingAcceptanceCriteria) // The "new AC" field
	fieldTags := fieldACEnd + 1
	fieldSession := fieldTags + 1
	fieldModelSize := fieldSession + 1
	fieldPriority := fieldModelSize + 1
	fieldBlockingReason := fieldPriority + 1
	fieldDependsOn := fieldBlockingReason + 1
	fieldSave := fieldDependsOn + 1

	// Number of options for selection fields
	numModelSizeOptions := 4      // (default), small, medium, large
	numPriorityOptions := 4       // low, medium, high, urgent
	numBlockingReasonOptions := 5 // (blank), Human needed, Waiting for dependency, Needs research, (custom)

	// Count real sessions (excluding pseudo-sessions)
	numSessionOptions := 1 // Start with "(none)"
	for _, sess := range m.sessions {
		if sess.ID != PseudoSessionAll && sess.ID != PseudoSessionUntagged {
			numSessionOptions++
		}
	}

	// Calculate the maximum field index (Save is the last field)
	maxFieldIndex := fieldSave

	// Helper to check if we're on a text input field
	isTextInputField := func(field int) bool {
		// Blocking reason field is text input only when custom option (4) is selected
		if field == fieldBlockingReason && m.pendingBallBlockingReason == 4 {
			return true
		}
		return field == fieldContext || field == fieldIntent || field == fieldTags ||
			(field >= fieldACStart && field <= fieldACEnd)
	}

	// Helper to check if we're on an AC field
	isACField := func(field int) bool {
		return field >= fieldACStart && field <= fieldACEnd
	}

	// Helper to check if we're on a field that supports @ file autocomplete
	// (Context, Title, and ACs - but NOT Tags)
	isAutocompleteField := func(field int) bool {
		return field == fieldContext || field == fieldIntent ||
			(field >= fieldACStart && field <= fieldACEnd)
	}

	// Helper to update autocomplete state after text changes
	updateAutocomplete := func() {
		if m.fileAutocomplete != nil && isAutocompleteField(m.pendingBallFormField) {
			text := m.textInput.Value()
			cursorPos := m.textInput.Position()
			m.fileAutocomplete.UpdateFromText(text, cursorPos)
		} else if m.fileAutocomplete != nil {
			m.fileAutocomplete.Reset()
		}
	}

	// Helper to save current field value before moving
	saveCurrentFieldValue := func() {
		switch m.pendingBallFormField {
		case fieldContext:
			// Get value from textarea for context field
			m.pendingBallContext = strings.TrimSpace(m.contextInput.Value())
		case fieldIntent:
			m.pendingBallIntent = strings.TrimSpace(m.textInput.Value())
		default:
			value := strings.TrimSpace(m.textInput.Value())
			// Check if it's Tags field (dynamic index)
			if m.pendingBallFormField == fieldTags {
				m.pendingBallTags = value
			} else if m.pendingBallFormField == fieldBlockingReason && m.pendingBallBlockingReason == 4 {
				// Custom blocking reason text
				m.pendingBallCustomReason = value
			} else if isACField(m.pendingBallFormField) {
				// AC field
				acIndex := m.pendingBallFormField - fieldACStart
				if acIndex < len(m.pendingAcceptanceCriteria) {
					// Editing existing AC
					if value == "" {
						// Remove empty ACs
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

	// Helper to recalculate dynamic field indices after AC changes
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

	// Helper to load field value into text input when entering field
	loadFieldValue := func(field int) {
		// Recalculate indices since ACs may have changed
		acEnd, tagsField, _, _, _, blockingReasonField, _, _ := recalcFieldIndices()

		m.textInput.Reset()
		switch field {
		case fieldContext:
			// Use textarea for context field
			m.contextInput.SetValue(m.pendingBallContext)
			m.contextInput.Focus()
			m.textInput.Blur()
			// Dynamically adjust height based on content
			adjustContextTextareaHeight(&m)
		case fieldIntent:
			m.contextInput.Blur()
			m.textInput.SetValue(m.pendingBallIntent)
			// Set placeholder - use context-derived placeholder if context has content and title is empty
			if m.pendingBallContext != "" && m.pendingBallIntent == "" {
				placeholder := generateTitlePlaceholderFromContext(m.pendingBallContext)
				if placeholder != "" {
					m.textInput.Placeholder = placeholder
				} else {
					m.textInput.Placeholder = "What is this ball about? (50 char recommended)"
				}
			} else {
				m.textInput.Placeholder = "What is this ball about? (50 char recommended)"
			}
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
				// Selection field
				m.textInput.Blur()
			}
		}
	}

	switch msg.String() {
	case "esc":
		// Cancel input - clear pending state
		m.clearPendingBallState()
		m.mode = splitView
		m.message = "Cancelled"
		m.textInput.Blur()
		m.contextInput.Blur()
		return m, nil

	case "ctrl+enter", "ctrl+s":
		// Create the ball (ctrl+s is more reliable across terminals)
		// Save current field value first
		saveCurrentFieldValue()

		// Validate required fields - title can be empty if context has content (will auto-generate)
		if m.pendingBallIntent == "" && m.pendingBallContext == "" {
			m.message = "Title is required (or add context to auto-generate)"
			return m, nil
		}

		return m.finalizeBallCreation()

	case "enter":
		// Check if we're navigating AC templates
		if m.acTemplateCursor >= 0 && m.acTemplateCursor < len(m.acTemplates) {
			// Add the selected template to ACs
			template := m.acTemplates[m.acTemplateCursor]
			if m.acTemplateSelected == nil {
				m.acTemplateSelected = make([]bool, len(m.acTemplates))
			}
			if !m.acTemplateSelected[m.acTemplateCursor] {
				// Only add if not already added
				m.pendingAcceptanceCriteria = append(m.pendingAcceptanceCriteria, template)
				m.acTemplateSelected[m.acTemplateCursor] = true
				m.message = "Added template: " + truncate(template, 30)
			} else {
				m.message = "Template already added"
			}
			// Move to next template or stay if at end
			m.acTemplateCursor++
			if m.acTemplateCursor >= len(m.acTemplates) {
				m.acTemplateCursor = len(m.acTemplates) - 1 // Stay at last
			}
			return m, nil
		}
		// Behavior depends on current field
		if m.pendingBallFormField == fieldContext {
			// For context field, Enter adds newline in textarea
			var cmd tea.Cmd
			m.contextInput, cmd = m.contextInput.Update(msg)
			adjustContextTextareaHeight(&m)
			// Update pendingBallContext live so title placeholder updates as you type
			m.pendingBallContext = m.contextInput.Value()
			return m, cmd
		} else if m.pendingBallFormField == fieldSave {
			// Save button - finalize ball creation
			saveCurrentFieldValue()
			// Validate required fields - title can be empty if context has content (will auto-generate)
			if m.pendingBallIntent == "" && m.pendingBallContext == "" {
				m.message = "Title is required (or add context to auto-generate)"
				return m, nil
			}
			return m.finalizeBallCreation()
		} else if m.pendingBallFormField == fieldDependsOn {
			// Open dependency selector
			return m.openDependencySelector()
		} else if isACField(m.pendingBallFormField) {
			acIndex := m.pendingBallFormField - fieldACStart
			value := strings.TrimSpace(m.textInput.Value())

			if acIndex == len(m.pendingAcceptanceCriteria) {
				// On the "new AC" field
				if value == "" {
					// Empty enter on the new AC field - create the ball
					saveCurrentFieldValue()
					// Validate required fields - title can be empty if context has content (will auto-generate)
					if m.pendingBallIntent == "" && m.pendingBallContext == "" {
						m.message = "Title is required (or add context to auto-generate)"
						return m, nil
					}
					return m.finalizeBallCreation()
				} else {
					// Add new AC and stay on the new AC field
					m.pendingAcceptanceCriteria = append(m.pendingAcceptanceCriteria, value)
					m.pendingNewAC = "" // Clear preserved content since it was added
					m.textInput.Reset()
					m.textInput.Placeholder = "New acceptance criterion (Enter on empty = save)"
					m.pendingBallFormField = fieldACStart + len(m.pendingAcceptanceCriteria) // Move to new "add" field
				}
			} else {
				// Editing existing AC - save and move to next field
				saveCurrentFieldValue()
				m.pendingBallFormField++
				// Recalculate indices after potential removal
				newACEnd, _, _, _, _, _, _, newSave := recalcFieldIndices()
				maxFieldIndex = newSave
				// Clamp to valid range
				if m.pendingBallFormField > newACEnd {
					// If we went past AC section, jump to Tags
					_, newFieldTags, _, _, _, _, _, _ := recalcFieldIndices()
					m.pendingBallFormField = newFieldTags
				}
				loadFieldValue(m.pendingBallFormField)
			}
		} else {
			// On other fields - save and move to next
			saveCurrentFieldValue()
			m.pendingBallFormField++
			// Recalculate after potential changes
			_, _, _, _, _, _, _, newSave := recalcFieldIndices()
			maxFieldIndex = newSave
			if m.pendingBallFormField > maxFieldIndex {
				m.pendingBallFormField = maxFieldIndex
			}
			loadFieldValue(m.pendingBallFormField)
		}
		return m, nil

	case "up":
		// If autocomplete is active, navigate suggestions instead of fields
		if m.fileAutocomplete != nil && m.fileAutocomplete.Active && len(m.fileAutocomplete.Suggestions) > 0 {
			m.fileAutocomplete.SelectPrev()
			return m, nil
		}
		// Check if we're navigating AC templates
		if m.acTemplateCursor >= 0 && len(m.acTemplates) > 0 {
			// We're in template navigation mode
			m.acTemplateCursor--
			if m.acTemplateCursor < 0 {
				// Exit template navigation, stay on new AC field
				m.acTemplateCursor = -1
			}
			return m, nil
		}
		// Arrow key up always moves to previous field
		saveCurrentFieldValue()
		m.pendingBallFormField--
		// Recalculate after potential removal
		_, _, _, _, _, _, _, newSave := recalcFieldIndices()
		maxFieldIndex = newSave
		if m.pendingBallFormField < 0 {
			m.pendingBallFormField = maxFieldIndex
		}
		loadFieldValue(m.pendingBallFormField)
		return m, nil

	case "down":
		// If autocomplete is active, navigate suggestions instead of fields
		if m.fileAutocomplete != nil && m.fileAutocomplete.Active && len(m.fileAutocomplete.Suggestions) > 0 {
			m.fileAutocomplete.SelectNext()
			return m, nil
		}
		// Check if we're navigating AC templates
		newACEnd, newFieldTags, _, _, _, _, _, newSave := recalcFieldIndices()
		if m.acTemplateCursor >= 0 && len(m.acTemplates) > 0 {
			// We're in template navigation mode
			m.acTemplateCursor++
			if m.acTemplateCursor >= len(m.acTemplates) {
				// Exit template navigation, move to Tags
				m.acTemplateCursor = -1
				saveCurrentFieldValue()
				m.pendingBallFormField = newFieldTags
				loadFieldValue(m.pendingBallFormField)
			}
			return m, nil
		}
		// Arrow key down always moves to next field
		saveCurrentFieldValue()
		// Check if we're on the "new AC" field - if so, enter template mode or move to Tags
		if m.pendingBallFormField == newACEnd {
			if len(m.acTemplates) > 0 {
				// Enter template selection mode
				m.acTemplateCursor = 0
				return m, nil
			}
			// No templates, move to Tags
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

	case "k":
		// k should ONLY be used for typing in text fields, never for navigation
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				adjustContextTextareaHeight(&m)
				// Update pendingBallContext live so title placeholder updates as you type
				m.pendingBallContext = m.contextInput.Value()
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		// On selection fields, k is just ignored (can't type in selection fields)
		return m, nil

	case "j":
		// j should ONLY be used for typing in text fields, never for navigation
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				adjustContextTextareaHeight(&m)
				// Update pendingBallContext live so title placeholder updates as you type
				m.pendingBallContext = m.contextInput.Value()
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		// On selection fields, j is just ignored (can't type in selection fields)
		return m, nil

	case "left":
		// Arrow key left only cycles selection left for selection fields
		_, _, sessionField, modelSizeField, priorityField, blockingReasonField, _, _ := recalcFieldIndices()
		if m.pendingBallFormField == sessionField {
			m.pendingBallSession--
			if m.pendingBallSession < 0 {
				m.pendingBallSession = numSessionOptions - 1
			}
			// Reload ACs when session changes
			m.loadACTemplatesAndRepoACs()
		} else if m.pendingBallFormField == modelSizeField {
			m.pendingBallModelSize--
			if m.pendingBallModelSize < 0 {
				m.pendingBallModelSize = numModelSizeOptions - 1
			}
		} else if m.pendingBallFormField == priorityField {
			m.pendingBallPriority--
			if m.pendingBallPriority < 0 {
				m.pendingBallPriority = numPriorityOptions - 1
			}
		} else if m.pendingBallFormField == blockingReasonField {
			// Cycle through blocking reason options
			m.pendingBallBlockingReason--
			if m.pendingBallBlockingReason < 0 {
				m.pendingBallBlockingReason = numBlockingReasonOptions - 1
			}
			// Load text input if switching to/from custom mode
			loadFieldValue(m.pendingBallFormField)
		}
		return m, nil

	case "right":
		// Arrow key right only cycles selection right for selection fields
		_, _, sessionField, modelSizeField, priorityField, blockingReasonField, _, _ := recalcFieldIndices()
		if m.pendingBallFormField == sessionField {
			m.pendingBallSession++
			if m.pendingBallSession >= numSessionOptions {
				m.pendingBallSession = 0
			}
			// Reload ACs when session changes
			m.loadACTemplatesAndRepoACs()
		} else if m.pendingBallFormField == modelSizeField {
			m.pendingBallModelSize++
			if m.pendingBallModelSize >= numModelSizeOptions {
				m.pendingBallModelSize = 0
			}
		} else if m.pendingBallFormField == priorityField {
			m.pendingBallPriority++
			if m.pendingBallPriority >= numPriorityOptions {
				m.pendingBallPriority = 0
			}
		} else if m.pendingBallFormField == blockingReasonField {
			// Cycle through blocking reason options
			m.pendingBallBlockingReason++
			if m.pendingBallBlockingReason >= numBlockingReasonOptions {
				m.pendingBallBlockingReason = 0
			}
			// Load text input if switching to/from custom mode
			loadFieldValue(m.pendingBallFormField)
		}
		return m, nil

	case "h":
		// h should ONLY be used for typing in text fields, never for navigation
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				adjustContextTextareaHeight(&m)
				// Update pendingBallContext live so title placeholder updates as you type
				m.pendingBallContext = m.contextInput.Value()
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		// On selection fields, h is just ignored (can't type in selection fields)
		return m, nil

	case "l":
		// l should ONLY be used for typing in text fields, never for navigation
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				adjustContextTextareaHeight(&m)
				// Update pendingBallContext live so title placeholder updates as you type
				m.pendingBallContext = m.contextInput.Value()
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		// On selection fields, l is just ignored (can't type in selection fields)
		return m, nil

	case "tab":
		// If autocomplete is active and we're on an autocomplete field, accept the completion
		if m.fileAutocomplete != nil && m.fileAutocomplete.Active && len(m.fileAutocomplete.Suggestions) > 0 {
			// Apply the selected completion
			if m.pendingBallFormField == fieldContext {
				newText := m.fileAutocomplete.ApplyCompletion(m.contextInput.Value())
				m.contextInput.SetValue(newText)
				adjustContextTextareaHeight(&m)
				// Update pendingBallContext live so title placeholder updates
				m.pendingBallContext = m.contextInput.Value()
			} else {
				newText := m.fileAutocomplete.ApplyCompletion(m.textInput.Value())
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
			// Reload ACs when session changes
			m.loadACTemplatesAndRepoACs()
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

	case "backspace", "delete":
		// Allow deletion in text fields
		// Note: Backspace doesn't re-trigger autocomplete (per AC requirement)
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				// Use textarea for context
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				// Adjust height after deletion
				adjustContextTextareaHeight(&m)
				// Update pendingBallContext live so title placeholder updates as you type
				m.pendingBallContext = m.contextInput.Value()
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			// Don't update autocomplete on backspace - only @ typing triggers it
			return m, cmd
		}
		return m, nil

	case " ":
		// Space dismisses autocomplete (per AC requirement)
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				// Use textarea for context
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				adjustContextTextareaHeight(&m)
				// Update pendingBallContext live so title placeholder updates as you type
				m.pendingBallContext = m.contextInput.Value()
				// Dismiss autocomplete on space
				if m.fileAutocomplete != nil {
					m.fileAutocomplete.Deactivate()
				}
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			// Dismiss autocomplete on space
			if m.fileAutocomplete != nil {
				m.fileAutocomplete.Deactivate()
			}
			return m, cmd
		}
		return m, nil

	default:
		// Pass to textinput only if on text input field
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				// Use textarea for context
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				// Adjust height after typing
				adjustContextTextareaHeight(&m)
				// Update pendingBallContext live so title placeholder updates as you type
				m.pendingBallContext = m.contextInput.Value()
				// Update autocomplete state after text changes (for @ detection)
				if m.fileAutocomplete != nil {
					text := m.contextInput.Value()
					cursorPos := m.contextInput.LineInfo().CharOffset
					m.fileAutocomplete.UpdateFromText(text, cursorPos)
				}
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			// Update autocomplete state after text changes (for @ detection)
			updateAutocomplete()
			return m, cmd
		}
		return m, nil
	}
}

// openDependencySelector opens the dependency selector for the ball form
func (m Model) openDependencySelector() (tea.Model, tea.Cmd) {
	// Build list of non-complete balls that can be dependencies
	m.dependencySelectBalls = make([]*session.Ball, 0)
	for _, ball := range m.balls {
		// Exclude complete/researched balls
		if ball.State != session.StateComplete && ball.State != session.StateResearched {
			m.dependencySelectBalls = append(m.dependencySelectBalls, ball)
		}
	}

	if len(m.dependencySelectBalls) == 0 {
		m.message = "No non-complete balls available as dependencies"
		return m, nil
	}

	// Initialize selection state from current pendingBallDependsOn
	m.dependencySelectActive = make(map[string]bool)
	for _, depID := range m.pendingBallDependsOn {
		m.dependencySelectActive[depID] = true
	}
	m.dependencySelectIndex = 0
	m.mode = dependencySelectorView
	return m, nil
}

// handleDependencySelectorKey handles keyboard input in the dependency selector view
func (m Model) handleDependencySelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Cancel selection - return to form without saving
		m.mode = unifiedBallFormView
		m.dependencySelectBalls = nil
		m.dependencySelectActive = nil
		m.message = "Cancelled"
		return m, nil

	case "up", "k":
		// Move selection up
		if m.dependencySelectIndex > 0 {
			m.dependencySelectIndex--
		}
		return m, nil

	case "down", "j":
		// Move selection down
		if m.dependencySelectIndex < len(m.dependencySelectBalls)-1 {
			m.dependencySelectIndex++
		}
		return m, nil

	case " ":
		// Toggle selection on current item
		if len(m.dependencySelectBalls) > 0 && m.dependencySelectIndex < len(m.dependencySelectBalls) {
			ball := m.dependencySelectBalls[m.dependencySelectIndex]
			if m.dependencySelectActive[ball.ID] {
				delete(m.dependencySelectActive, ball.ID)
			} else {
				m.dependencySelectActive[ball.ID] = true
			}
		}
		return m, nil

	case "enter":
		// Confirm selection - save to pendingBallDependsOn and return to form
		m.pendingBallDependsOn = make([]string, 0)
		for ballID := range m.dependencySelectActive {
			m.pendingBallDependsOn = append(m.pendingBallDependsOn, ballID)
		}
		// Sort for consistent display
		sort.Strings(m.pendingBallDependsOn)

		m.mode = unifiedBallFormView
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
