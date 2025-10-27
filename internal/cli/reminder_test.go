package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ohare93/juggle/internal/session"
)

func TestReminderCommand(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		setupMarker    bool
		markerAge      time.Duration
		expectReminder bool
	}{
		{
			name:           "No marker file - should show reminder",
			setupMarker:    false,
			expectReminder: true,
		},
		{
			name:           "Fresh marker file - should not show reminder",
			setupMarker:    true,
			markerAge:      1 * time.Minute,
			expectReminder: false,
		},
		{
			name:           "Old marker file - should show reminder",
			setupMarker:    true,
			markerAge:      10 * time.Minute,
			expectReminder: true,
		},
		{
			name:           "Marker at threshold boundary - should not show reminder",
			setupMarker:    true,
			markerAge:      4*time.Minute + 59*time.Second,
			expectReminder: false,
		},
		{
			name:           "Marker just past threshold - should show reminder",
			setupMarker:    true,
			markerAge:      5*time.Minute + 1*time.Second,
			expectReminder: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup marker file if needed
			if tt.setupMarker {
				markerPath := session.GetMarkerFilePathForTest(tempDir)

				// Create marker file
				marker := session.MarkerFile{
					Timestamp: time.Now().Add(-tt.markerAge),
					Project:   tempDir,
				}

				if err := session.WriteMarkerFileForTest(markerPath, marker); err != nil {
					t.Fatalf("Failed to create marker file: %v", err)
				}

				// Update file's mtime to match the age
				if err := os.Chtimes(markerPath, time.Now(), time.Now().Add(-tt.markerAge)); err != nil {
					t.Fatalf("Failed to set marker file time: %v", err)
				}
			}

			// Check the actual function directly (simpler than mocking working directory)
			shouldShow, err := session.ShouldShowReminder(tempDir)
			if err != nil {
				t.Fatalf("ShouldShowReminder failed: %v", err)
			}

			// Verify expectation
			if tt.expectReminder {
				if !shouldShow {
					t.Errorf("Expected reminder to show, but ShouldShowReminder returned false")
				}
			} else {
				if shouldShow {
					t.Errorf("Expected no reminder, but ShouldShowReminder returned true")
				}
			}

			// Clean up marker file for next test
			markerPath := session.GetMarkerFilePathForTest(tempDir)
			os.Remove(markerPath)
		})
	}
}

func TestUpdateCheckMarker(t *testing.T) {
	tempDir := t.TempDir()
	markerPath := session.GetMarkerFilePathForTest(tempDir)

	// Ensure no marker exists
	os.Remove(markerPath)

	// Update marker
	err := session.UpdateCheckMarker(tempDir)
	if err != nil {
		t.Errorf("UpdateCheckMarker returned error: %v", err)
	}

	// Check marker was created
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("Marker file was not created")
	}

	// Check marker is recent
	info, err := os.Stat(markerPath)
	if err != nil {
		t.Fatalf("Failed to stat marker file: %v", err)
	}

	age := time.Since(info.ModTime())
	if age > 5*time.Second {
		t.Errorf("Marker file is too old: %v", age)
	}

	// Verify no reminder should be shown
	shouldShow, err := session.ShouldShowReminder(tempDir)
	if err != nil {
		t.Fatalf("ShouldShowReminder failed: %v", err)
	}
	if shouldShow {
		t.Error("Reminder should not be shown after updating marker")
	}
}

func TestMarkerFileConsistency(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Two different directories should have different marker files
	marker1 := session.GetMarkerFilePathForTest(tempDir1)
	marker2 := session.GetMarkerFilePathForTest(tempDir2)

	if marker1 == marker2 {
		t.Error("Different directories should have different marker files")
	}

	// Same directory should always get the same marker file
	marker1a := session.GetMarkerFilePathForTest(tempDir1)
	if marker1 != marker1a {
		t.Error("Same directory should always get the same marker file path")
	}
}

func TestMarkerFileLocation(t *testing.T) {
	tempDir := t.TempDir()
	markerPath := session.GetMarkerFilePathForTest(tempDir)

	// Verify marker is in /tmp
	if !filepath.IsAbs(markerPath) {
		t.Error("Marker path should be absolute")
	}

	if filepath.Dir(markerPath) != "/tmp" {
		t.Errorf("Marker should be in /tmp, got: %s", filepath.Dir(markerPath))
	}

	// Verify filename format
	basename := filepath.Base(markerPath)
	if len(basename) < len("juggle-check-") {
		t.Error("Marker filename too short")
	}

	prefix := basename[:len("juggle-check-")]
	if prefix != "juggle-check-" {
		t.Errorf("Marker filename should start with 'juggle-check-', got: %s", prefix)
	}
}

func TestReminderPerformance(t *testing.T) {
	tempDir := t.TempDir()

	// Update marker
	_ = session.UpdateCheckMarker(tempDir)

	// Measure performance of ShouldShowReminder
	start := time.Now()
	for i := 0; i < 100; i++ {
		_, _ = session.ShouldShowReminder(tempDir)
	}
	duration := time.Since(start)

	// Should complete 100 checks in well under 50ms total
	if duration > 50*time.Millisecond {
		t.Errorf("ShouldShowReminder too slow: 100 checks took %v (should be < 50ms)", duration)
	}

	// Average should be well under 0.5ms per check
	avgPerCheck := duration / 100
	if avgPerCheck > 500*time.Microsecond {
		t.Logf("Average check time: %v (target: < 500µs)", avgPerCheck)
	}
}

func TestReminderErrorHandling(t *testing.T) {
	// Test with non-existent directory - should not panic
	_, err := session.ShouldShowReminder("/this/path/does/not/exist/hopefully")
	if err != nil {
		// Should silently handle errors
		t.Logf("Got expected error for non-existent path: %v", err)
	}

	// Update marker for non-existent directory - should not panic
	err = session.UpdateCheckMarker("/this/path/does/not/exist/hopefully")
	if err != nil {
		// Should silently handle errors
		t.Logf("Got expected error for non-existent path: %v", err)
	}

	// Test with empty path - should not panic
	_, err = session.ShouldShowReminder("")
	if err != nil {
		t.Logf("Got expected error for empty path: %v", err)
	}
}
