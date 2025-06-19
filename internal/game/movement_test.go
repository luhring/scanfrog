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
			expectedRow: 22, // topMargin(0) + header(1) + separator(1) + finish(1) + empty(1) + hint(1) + rows 3-18 + frog at row 19 = 22
		},
		{
			name:        "move up once",
			moves:       []string{"up"},
			expectedY:   18,
			expectedRow: 21,
		},
		{
			name:        "move to top road lane",
			moves:       []string{"up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up"},
			expectedY:   4,
			expectedRow: 7, // topMargin(0) + header(1) + separator(1) + finish(1) + empty(1) + hint(1) + empty(1) + road at row 4 = 7
		},
		{
			name:        "move to row above top road lane",
			moves:       []string{"up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up", "up"},
			expectedY:   3,
			expectedRow: 6, // The empty row above top road lane
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
					t.Logf("%s Row %2d: %s", marker, i, line[:minInt(40, len(line))])
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

// TestRow3NotSkipped verifies that row 3 (empty row above top road lane) is visually distinct
// This test verifies that row 3 is truly empty and not occupied by road lanes
func TestRow3NotSkipped(t *testing.T) {
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.width = 80
	model.height = 24
	gameModel := model.startGame(vulns)

	// Move frog to row 3 (should be empty row above top road lane)
	gameModel.frog.y = 3
	gameModel.hasMoved = true
	gameModel.firstMoveTime = time.Now().Add(-2 * time.Second) // Ensure hint is gone

	// Verify no lanes at row 3
	hasLaneAtRow3 := false
	for _, lane := range gameModel.lanes {
		if lane.y == 3 {
			hasLaneAtRow3 = true
			break
		}
	}

	if hasLaneAtRow3 {
		t.Error("Row 3 should be empty but has a road lane!")
	}

	// Render and analyze
	output := gameModel.renderGame()
	lines := strings.Split(output, "\n")

	// Find the frog
	frogRow := -1
	for i, line := range lines {
		if strings.Contains(line, "🐸") {
			frogRow = i
			break
		}
	}

	// Verify the frog's visual row doesn't contain road markings
	if frogRow >= 0 && frogRow < len(lines) {
		if strings.Contains(lines[frogRow], "━") {
			t.Error("Row 3 is being visually merged with a road lane!")
			t.Logf("Frog at visual row %d contains road markings: %s", frogRow, lines[frogRow][:min(40, len(lines[frogRow]))])
		}
	}

	// Log the game state for debugging
	t.Logf("Frog position: y=%d, visual row=%d", gameModel.frog.y, frogRow)
	t.Logf("Lanes: %v", func() []int {
		result := make([]int, len(gameModel.lanes))
		for i, lane := range gameModel.lanes {
			result[i] = lane.y
		}
		return result
	}())
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

	// Send movement commands
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})

	// Send quit command
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	// Wait for the test to finish
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Get final output after the test has finished
	outputReader := tm.FinalOutput(t)
	outputBytes, _ := io.ReadAll(outputReader)
	output := string(outputBytes)
	if !strings.Contains(output, "FINISH") {
		t.Error("Game didn't render finish line")
	}
}

// TestNoRowSkippingFromBottom verifies that moving up from bottom empty row doesn't skip the road
func TestNoRowSkippingFromBottom(t *testing.T) {
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.width = 80
	model.height = 24
	gameModel := model.startGame(vulns)

	// Debug: log all lanes
	t.Logf("All lanes: %v", func() []int {
		result := make([]int, len(gameModel.lanes))
		for i, lane := range gameModel.lanes {
			result[i] = lane.y
		}
		return result
	}())

	// Start at bottom (y=19)
	gameModel.frog.y = 19

	// Move up once
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("up")}
	newModel, _ := gameModel.handleKeyPress(msg)
	gameModel = newModel.(Model)

	// Should be at y=18
	if gameModel.frog.y != 18 {
		t.Errorf("Frog skipped a row! Expected y=18, got y=%d", gameModel.frog.y)
	}

	// Based on the screenshot, there should NOT be a lane at y=17
	// The bottom lane should be higher up, creating the gap we see
	hasLaneAt17 := false
	for _, lane := range gameModel.lanes {
		if lane.y == 17 {
			hasLaneAt17 = true
			break
		}
	}

	if hasLaneAt17 {
		t.Error("There should NOT be a road lane at y=17 - this creates the skipping issue!")
	}

	// Find the bottom-most lane
	bottomLaneY := -1
	for _, lane := range gameModel.lanes {
		if bottomLaneY == -1 || lane.y > bottomLaneY {
			bottomLaneY = lane.y
		}
	}

	t.Logf("Bottom lane is at y=%d", bottomLaneY)
	if bottomLaneY == 17 {
		t.Error("Bottom lane at y=17 means frog will skip from y=18 to y=16 when moving up!")
	}
}

// TestExactlyThreeRowsBetweenTopRoadAndFinish verifies spacing at top of board
func TestExactlyThreeRowsBetweenTopRoadAndFinish(t *testing.T) {
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.width = 80
	model.height = 24
	gameModel := model.startGame(vulns)

	// Find the topmost road lane
	topRoadY := -1
	for _, lane := range gameModel.lanes {
		if topRoadY == -1 || lane.y < topRoadY {
			topRoadY = lane.y
		}
	}

	// Finish line is at y=0
	// Between finish (0) and top road, we should have exactly 3 rows:
	// Row 1: empty
	// Row 2: hint/empty
	// Row 3: empty
	// Row 4: top road (should be at y=4)

	expectedTopRoadY := 4
	if topRoadY != expectedTopRoadY {
		t.Errorf("Top road lane should be at y=%d but is at y=%d", expectedTopRoadY, topRoadY)
		t.Logf("This means there are %d rows between finish and top road, not 3", topRoadY-1)
	}

	// Also verify no lanes exist at rows 1, 2, or 3
	for _, lane := range gameModel.lanes {
		if lane.y >= 1 && lane.y <= 3 {
			t.Errorf("Found road lane at y=%d, but rows 1-3 should be empty!", lane.y)
		}
	}
}

// TestNoConsecutiveEmptyRows verifies there are no two empty rows in a row (except at top)
func TestNoConsecutiveEmptyRows(t *testing.T) {
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.width = 80
	model.height = 24
	gameModel := model.startGame(vulns)

	// Create a map of which rows have lanes
	hasLane := make(map[int]bool)
	for _, lane := range gameModel.lanes {
		hasLane[lane.y] = true
	}

	// Check for consecutive empty rows (excluding the top 4 rows which are special)
	for y := 19; y >= 5; y-- {
		if !hasLane[y] && !hasLane[y-1] {
			t.Errorf("Found consecutive empty rows at y=%d and y=%d", y, y-1)
		}
	}

	// Log the pattern for debugging
	t.Log("Row pattern from bottom to top:")
	for y := 19; y >= 0; y-- {
		if hasLane[y] {
			t.Logf("Row %2d: ROAD", y)
		} else {
			t.Logf("Row %2d: empty", y)
		}
	}
}

// TestExactLayoutPattern verifies the exact expected layout
func TestExactLayoutPattern(t *testing.T) {
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.width = 80
	model.height = 24
	gameModel := model.startGame(vulns)

	// Define expected layout from bottom to top
	expectedLayout := map[int]string{
		19: "empty", // Frog start
		18: "road",
		17: "empty",
		16: "road",
		15: "empty",
		14: "road",
		13: "empty",
		12: "road",
		11: "empty",
		10: "road",
		9:  "empty",
		8:  "road",
		7:  "empty",
		6:  "road",
		5:  "empty",
		4:  "road",  // Top road lane
		3:  "empty", // Empty row above top road
		2:  "empty", // Hint row
		1:  "empty", // Empty row below finish
		0:  "empty", // Finish line (not a road)
	}

	// Check actual layout
	for y := 0; y < 20; y++ {
		hasLane := false
		for _, lane := range gameModel.lanes {
			if lane.y == y {
				hasLane = true
				break
			}
		}

		expected := expectedLayout[y]
		actual := "empty"
		if hasLane {
			actual = "road"
		}

		if actual != expected {
			t.Errorf("Row %d: expected %s, got %s", y, expected, actual)
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestVisualRendering verifies the actual visual output matches expectations
func TestVisualRendering(t *testing.T) {
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.width = 80
	model.height = 24
	gameModel := model.startGame(vulns)

	// Hide hint to simplify counting
	gameModel.hasMoved = true
	gameModel.firstMoveTime = time.Now().Add(-2 * time.Second)

	// Render the game
	output := gameModel.renderGame()
	lines := strings.Split(output, "\n")

	// Count all lines for debugging
	t.Logf("Total lines in output: %d", len(lines))

	// Find key landmarks
	finishLine := -1
	topRoadLine := -1

	for i, line := range lines {
		if strings.Contains(line, "FINISH") {
			finishLine = i
			t.Logf("Found FINISH at line %d", i)
		}
		if strings.Contains(line, "━") && topRoadLine == -1 {
			topRoadLine = i
			t.Logf("Found top road at line %d", i)
		}
		if strings.Contains(line, "🐸") {
			t.Logf("Found frog at line %d", i)
		}
	}

	// Count rows between finish and top road
	if finishLine >= 0 && topRoadLine >= 0 {
		rowsBetween := topRoadLine - finishLine - 1
		t.Logf("Rows between finish line and top road: %d", rowsBetween)
		if rowsBetween != 3 {
			t.Errorf("Expected 3 rows between finish and top road, got %d", rowsBetween)
		}
	}

	// Print the visual layout for debugging
	t.Log("Visual layout:")
	for i, line := range lines {
		marker := "  "
		switch {
		case i == finishLine:
			marker = "F "
		case i == topRoadLine:
			marker = "R "
		case strings.Contains(line, "━"):
			marker = "r "
		}
		t.Logf("%s Line %2d: %s", marker, i, line[:minInt(60, len(line))])
	}

	// Check for consecutive empty rows
	for i := 1; i < len(lines)-1; i++ {
		currentEmpty := !strings.Contains(lines[i], "━") && !strings.Contains(lines[i], "FINISH") && !strings.Contains(lines[i], "🐸")
		nextEmpty := !strings.Contains(lines[i+1], "━") && !strings.Contains(lines[i+1], "FINISH") && !strings.Contains(lines[i+1], "🐸")

		if currentEmpty && nextEmpty && i > topRoadLine {
			t.Logf("Found consecutive empty rows at lines %d and %d", i, i+1)
		}
	}
}

// TestActualRowCounting counts the exact rendering output
func TestActualRowCounting(t *testing.T) {
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.width = 80
	model.height = 24
	gameModel := model.startGame(vulns)

	// Test with hint visible
	gameModel.hasMoved = false
	output := gameModel.renderGame()
	lines := strings.Split(output, "\n")

	// Count line types
	headerCount := 0
	roadCount := 0
	emptyCount := 0
	finishCount := 0
	hintCount := 0

	for i, line := range lines {
		switch {
		case strings.Contains(line, "scanfrog"):
			headerCount++
			t.Logf("Line %d: HEADER", i)
		case strings.Contains(line, "───") || strings.Contains(line, "─") && i == 1:
			headerCount++ // separator
			t.Logf("Line %d: SEPARATOR", i)
		case strings.Contains(line, "FINISH"):
			finishCount++
			t.Logf("Line %d: FINISH", i)
		case strings.Contains(line, "━"):
			roadCount++
			t.Logf("Line %d: ROAD", i)
		case strings.Contains(line, "Make it to the finish"):
			hintCount++
			t.Logf("Line %d: HINT", i)
		case strings.TrimSpace(line) == "":
			emptyCount++
			t.Logf("Line %d: EMPTY (blank)", i)
		default:
			emptyCount++
			t.Logf("Line %d: EMPTY (with content: %s)", i, strings.TrimSpace(line)[:minInt(30, len(strings.TrimSpace(line)))])
		}
	}

	t.Logf("Total lines: %d", len(lines))
	t.Logf("Header lines: %d", headerCount)
	t.Logf("Road lines: %d", roadCount)
	t.Logf("Empty lines: %d", emptyCount)
	t.Logf("Finish lines: %d", finishCount)
	t.Logf("Hint lines: %d", hintCount)
}

// TestFrogMovementNoSkipping verifies frog moves exactly one row at a time
func TestFrogMovementNoSkipping(t *testing.T) {
	vulns := []grype.Vulnerability{}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.width = 80
	model.height = 24
	gameModel := model.startGame(vulns)

	// Start at bottom (y=19)
	gameModel.frog.y = 19
	gameModel.hasMoved = true
	gameModel.firstMoveTime = time.Now().Add(-2 * time.Second)

	// Track all positions visited
	positionsVisited := []int{19}

	// Move up one row at a time until we reach the top
	for i := 0; i < 19; i++ {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("up")}
		newModel, _ := gameModel.handleKeyPress(msg)
		gameModel = newModel.(Model)
		positionsVisited = append(positionsVisited, gameModel.frog.y)
	}

	// Check that we visited every row from 19 down to 0
	expectedPositions := make([]int, 20)
	for i := 0; i < 20; i++ {
		expectedPositions[i] = 19 - i
	}

	for i, expected := range expectedPositions {
		if i < len(positionsVisited) {
			if positionsVisited[i] != expected {
				t.Errorf("Move %d: expected y=%d, got y=%d (skipped a row!)", i, expected, positionsVisited[i])
			}
		}
	}

	// Ensure we can reach the finish line
	if gameModel.frog.y != 0 {
		t.Errorf("Failed to reach finish line, stuck at y=%d", gameModel.frog.y)
	}

	// Log the movement pattern for debugging
	t.Logf("Movement pattern: %v", positionsVisited)
}
