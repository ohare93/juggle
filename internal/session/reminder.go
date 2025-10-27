package session

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MarkerFile represents the structure of the reminder marker file
type MarkerFile struct {
	Timestamp time.Time `json:"timestamp"`
	Project   string    `json:"project"`
}

// ReminderThreshold is how long before showing reminder again (5 minutes)
const ReminderThreshold = 5 * time.Minute

// Test helpers - exported for testing only

// GetMarkerFilePathForTest returns the marker file path for testing
func GetMarkerFilePathForTest(workingDir string) string {
	return getMarkerFilePath(workingDir)
}

// WriteMarkerFileForTest writes a marker file for testing
func WriteMarkerFileForTest(path string, marker MarkerFile) error {
	data, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// getProjectHash returns a SHA256 hash of the working directory
// This provides a stable, filesystem-safe identifier for the project
func getProjectHash(workingDir string) string {
	hash := sha256.Sum256([]byte(workingDir))
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes (16 hex chars)
}

// getMarkerFilePath returns the path to the marker file for a project
// Format: /tmp/juggle-check-<hash>
func getMarkerFilePath(workingDir string) string {
	hash := getProjectHash(workingDir)
	return filepath.Join("/tmp", fmt.Sprintf("juggle-check-%s", hash))
}

// UpdateCheckMarker creates or updates the marker file for the current project
// This should be called whenever the user checks juggling state
// Returns nil on success, silently ignores errors (for hook safety)
func UpdateCheckMarker(workingDir string) error {
	markerPath := getMarkerFilePath(workingDir)

	marker := MarkerFile{
		Timestamp: time.Now(),
		Project:   workingDir,
	}

	data, err := json.Marshal(marker)
	if err != nil {
		// Silently ignore - hook safety
		return nil
	}

	// Write atomically by writing to temp file then renaming
	tempPath := markerPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		// Silently ignore - hook safety
		return nil
	}

	if err := os.Rename(tempPath, markerPath); err != nil {
		// Clean up temp file on error
		_ = os.Remove(tempPath)
		// Silently ignore - hook safety
		return nil
	}

	return nil
}

// ShouldShowReminder checks if a reminder should be shown
// Returns true if:
// - Marker file doesn't exist
// - Marker file is older than ReminderThreshold
// Always returns false on error (hook safety)
func ShouldShowReminder(workingDir string) (bool, error) {
	markerPath := getMarkerFilePath(workingDir)

	// Check if file exists
	info, err := os.Stat(markerPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - show reminder
			return true, nil
		}
		// Other error - silently don't show reminder
		return false, nil
	}

	// Check if file is too old
	age := time.Since(info.ModTime())
	if age > ReminderThreshold {
		return true, nil
	}

	// Recently checked - no reminder needed
	return false, nil
}
