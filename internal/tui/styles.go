package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Base styles
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		MarginBottom(1)

	ballStyle = lipgloss.NewStyle().
		Padding(0, 1)

	selectedBallStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("240")).
		Bold(true)

	// State colors
	readyColor      = lipgloss.Color("2")   // Green
	jugglingColor   = lipgloss.Color("3")   // Yellow
	droppedColor    = lipgloss.Color("1")   // Red
	completeColor   = lipgloss.Color("8")   // Gray
	researchedColor = lipgloss.Color("12")  // Light blue - for research tasks
	onHoldColor     = lipgloss.Color("5")   // Magenta - for deferred/on-hold tasks

	// Priority colors
	urgentColor = lipgloss.Color("9") // Bright red
	highColor   = lipgloss.Color("3") // Yellow
	mediumColor = lipgloss.Color("6") // Cyan
	lowColor    = lipgloss.Color("8") // Gray

	messageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).
		Bold(true)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
)
