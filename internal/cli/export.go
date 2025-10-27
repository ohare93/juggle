package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	exportFormat    string
	exportOutput    string
	exportIncludeDone bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export balls to JSON or CSV",
	Long: `Export session data to JSON or CSV format for analysis or backup.

By default exports all active balls (excluding done). Use --include-done to include archived balls.

Examples:
  juggler export --format json --output balls.json
  juggler export --format csv --output balls.csv --include-done
  juggler export --format json  # Outputs to stdout`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Export format: json or csv")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (default: stdout)")
	exportCmd.Flags().BoolVar(&exportIncludeDone, "include-done", false, "Include archived (done) balls in export")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Validate format
	if exportFormat != "json" && exportFormat != "csv" {
		return fmt.Errorf("invalid format: %s (must be json or csv)", exportFormat)
	}

	// Load config to discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Discover all projects
	projects, err := session.DiscoverProjects(config)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	if len(projects) == 0 {
		return fmt.Errorf("no projects with .juggler directories found")
	}

	// Load all balls
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	// Filter based on --include-done
	var balls []*session.Session
	if exportIncludeDone {
		balls = allBalls
	} else {
		balls = make([]*session.Session, 0)
		for _, ball := range allBalls {
			if ball.ActiveState != session.ActiveComplete {
				balls = append(balls, ball)
			}
		}
	}

	if len(balls) == 0 {
		return fmt.Errorf("no balls to export")
	}

	// Export based on format
	var output []byte
	switch exportFormat {
	case "json":
		output, err = exportJSON(balls)
	case "csv":
		output, err = exportCSV(balls)
	}

	if err != nil {
		return fmt.Errorf("failed to export: %w", err)
	}

	// Write to file or stdout
	if exportOutput != "" {
		if err := os.WriteFile(exportOutput, output, 0644); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
		fmt.Printf("✓ Exported %d ball(s) to %s\n", len(balls), exportOutput)
	} else {
		fmt.Print(string(output))
	}

	return nil
}

func exportJSON(balls []*session.Session) ([]byte, error) {
	// Create export structure
	export := struct {
		ExportedAt string             `json:"exported_at"`
		TotalBalls int                `json:"total_balls"`
		Balls      []*session.Session `json:"balls"`
	}{
		ExportedAt: fmt.Sprintf("%d", 1),
		TotalBalls: len(balls),
		Balls:      balls,
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, err
	}

	return data, nil
}

func exportCSV(balls []*session.Session) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"ID",
		"Project",
		"Intent",
		"Priority",
		"ActiveState",
		"JuggleState",
		"StartedAt",
		"CompletedAt",
		"LastActivity",
		"Tags",
		"TodosTotal",
		"TodosCompleted",
		"CompletionNote",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write rows
	for _, ball := range balls {
		completedAt := ""
		if ball.CompletedAt != nil {
			completedAt = ball.CompletedAt.Format("2006-01-02 15:04:05")
		}

		tags := strings.Join(ball.Tags, ";")

		total, completed := ball.TodoStats()

		juggleState := ""
		if ball.JuggleState != nil {
			juggleState = string(*ball.JuggleState)
		}
		row := []string{
			ball.ID,
			ball.WorkingDir,
			ball.Intent,
			string(ball.Priority),
			string(ball.ActiveState),
			juggleState,
			ball.StartedAt.Format("2006-01-02 15:04:05"),
			completedAt,
			ball.LastActivity.Format("2006-01-02 15:04:05"),
			tags,
			fmt.Sprintf("%d", total),
			fmt.Sprintf("%d", completed),
			ball.CompletionNote,
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}
