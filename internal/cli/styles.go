package cli

import "github.com/charmbracelet/lipgloss"

// Consistent color scheme for ball states across all views
var (
	// Juggle states (sub-states within juggling)
	StyleNeedsCaught = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red - needs user attention
	StyleNeedsThrown = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow - needs user input
	StyleInAir       = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green - actively working

	// Active states (top-level states)
	StyleJuggling = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green - active
	StyleReady    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue - planned/ready
	StyleDropped  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red - dropped
	StyleComplete = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // Gray - archived

	// Priority levels
	StyleUrgent = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)  // Red bold - urgent
	StyleHigh   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))             // Red - high
	StyleMedium = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))            // Yellow - medium
	StyleLow    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // Gray - low

	// UI elements
	StyleProject   = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))                                       // Cyan
	StyleDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))                                        // Gray
	StyleHighlight = lipgloss.NewStyle().Bold(true)                                                             // Bold
	StyleHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("8"))
)

// GetPriorityStyle returns the appropriate style for a given priority level
func GetPriorityStyle(priority string) lipgloss.Style {
	switch priority {
	case "urgent":
		return StyleUrgent
	case "high":
		return StyleHigh
	case "medium":
		return StyleMedium
	case "low":
		return StyleLow
	default:
		return lipgloss.NewStyle()
	}
}
