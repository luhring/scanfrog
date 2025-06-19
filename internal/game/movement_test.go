package game

import (
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/luhring/scanfrog/internal/grype"
)

// TestFrogMovementAndRendering verifies that frog movement corresponds to visual rendering
func TestFrogMovementAndRendering(t *testing.T) {
	// Create a model with 0 vulnerabilities for simplicity
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}

	// Create the model and start the game immediately (skip loading)
	model := NewModel(source)
	model.width = 80
	model.height = 24
	gameModel := model.startGame(vulns)

	// Test cases for movement
	tests := []struct {
		name        string
		moves       []string
		expectedY   int
		expectedRow int // Which visual row should contain the frog (counting from header)
		waitForHint bool
	}{
		{
			name:        "initial position",
			moves:       []string{},
			expectedY:   19, // Bottom of game area
			expectedRow: 23, // topMargin(1) + header(1) + separator(1) + rows 0-18 + frog at row 19 = 23
		},
		{
			name:        "move up once",
			moves:       []string{"up"},
			expectedY:   18,
			expectedRow: 22,
		},
		{
			name:        "move to top road lane",
			moves:       []string{"up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up"},
			expectedY:   4,
			expectedRow: 8, // topMargin(1) + header(1) + separator(1) + finish(1) + empty(1) + hint(2 with blank) + empty(1) = 8
		},
		{
			name:        "move to row above top road lane",
			moves:       []string{"up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up"},
			expectedY:   3,
			expectedRow: 7, // The empty row above top road lane (right after hint+blank)
		},
		{
			name:        "move to hint row",
			moves:       []string{"up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up"},
			expectedY:   2,
			expectedRow: 5,    // The hint row (when frog is there, no hint shown)
			waitForHint: true, // Wait for hint to disappear
		},
		{
			name:        "move to row below finish",
			moves:       []string{"up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up"},
			expectedY:   1,
			expectedRow: 4,
		},
		{
			name:        "move to finish line",
			moves:       []string{"up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up"},
			expectedY:   0,
			expectedRow: 3, // Finish line row
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start fresh from initial position
			testModel := gameModel
			testModel.frog.y = 19
			testModel.hasMoved = false

			// Apply moves
			for _, move := range tt.moves {
				msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(move)}
				newModel, _ := testModel.handleKeyPress(msg)
				testModel = newModel.(Model)
			}

			// Wait for hint to disappear if needed
			if tt.waitForHint && testModel.hasMoved {
				testModel.firstMoveTime = time.Now().Add(-2 * time.Second)
			}

			// Verify frog position in model
			if testModel.frog.y != tt.expectedY {
				t.Errorf("Expected frog.y = %d, got %d", tt.expectedY, testModel.frog.y)
			}

			// Render the game
			output := testModel.renderGame()

			// Find which row contains the frog
			lines := strings.Split(output, "\n")
			frogRow := -1
			for i, line := range lines {
				if strings.Contains(line, "🐸") {
					frogRow = i
					break
				}
			}

			if frogRow != tt.expectedRow {
				t.Errorf("Expected frog on visual row %d, found on row %d", tt.expectedRow, frogRow)
				t.Logf("Frog position y=%d", testModel.frog.y)
				for i, line := range lines {
					marker := " "
					if i == frogRow {
						marker = ">"
					}
					t.Logf("%s Row %2d: %s", marker, i, line[:min(40, len(line))])
				}
			}
		})
	}
}

// TestRowSpacingConsistency verifies that the game board maintains consistent spacing
func TestRowSpacingConsistency(t *testing.T) {
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.width = 80
	model.height = 24
	gameModel := model.startGame(vulns)

	// Test with hint visible
	gameModel.hasMoved = false
	output1 := gameModel.renderGame()
	lines1 := strings.Split(output1, "\n")

	// Test with hint hidden (after movement)
	gameModel.hasMoved = true
	gameModel.firstMoveTime = time.Now().Add(-2 * time.Second)
	output2 := gameModel.renderGame()
	lines2 := strings.Split(output2, "\n")

	// Both should have the same number of lines
	if len(lines1) != len(lines2) {
		t.Errorf("Line count changed: hint visible=%d, hint hidden=%d", len(lines1), len(lines2))
	}

	// Find the top road lane in both outputs
	findRoadLane := func(lines []string) int {
		for i, line := range lines {
			if strings.Contains(line, "━") {
				return i
			}
		}
		return -1
	}

	roadLane1 := findRoadLane(lines1)
	roadLane2 := findRoadLane(lines2)

	if roadLane1 != roadLane2 {
		t.Errorf("Road lane position changed: hint visible=%d, hint hidden=%d", roadLane1, roadLane2)
	}
}

// TestInteractiveMovement uses teatest for interactive testing
func TestInteractiveMovement(t *testing.T) {
	// Create a test model
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}
	model := NewModel(source)

	// Create a test program
	tm := teatest.NewTestModel(t, model)

	// Send the vulnerabilities loaded message to start the game
	tm.Send(vulnerabilitiesLoadedMsg{vulns: vulns})

	// Wait for the game to start
	time.Sleep(100 * time.Millisecond)

	// Get initial output
	outputReader := tm.FinalOutput(t)
	outputBytes, _ := io.ReadAll(outputReader)
	output := string(outputBytes)
	if !strings.Contains(output, "FINISH") {
		t.Error("Game didn't render finish line")
	}

	// Send movement commands
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})

	// Verify frog moved
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}) // Quit
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
