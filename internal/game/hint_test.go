// Package game provides the game logic and rendering for the Scanfrog vulnerability visualization game.
package game

import (
	"strings"
	"testing"
	"time"

	"github.com/luhring/scanfrog/internal/grype"
)

func TestHintDisplay(t *testing.T) {
	tests := []struct {
		name           string
		vulns          []grype.Vulnerability
		hasMoved       bool
		timeSinceMoved time.Duration
		expectHint     bool
		expectedText   string
	}{
		{
			name:         "zero vulns - no movement",
			vulns:        []grype.Vulnerability{},
			hasMoved:     false,
			expectHint:   true,
			expectedText: "Ahhh, so peaceful! (And boring!) Proceed to the finish line to win!",
		},
		{
			name: "has vulns - no movement",
			vulns: []grype.Vulnerability{
				{ID: "CVE-2021-1", Severity: "High"},
			},
			hasMoved:     false,
			expectHint:   true,
			expectedText: "Make it to the finish line without getting hit by anything!",
		},
		{
			name:           "zero vulns - moved recently",
			vulns:          []grype.Vulnerability{},
			hasMoved:       true,
			timeSinceMoved: 500 * time.Millisecond,
			expectHint:     true,
			expectedText:   "Ahhh, so peaceful! (And boring!) Proceed to the finish line to win!",
		},
		{
			name: "has vulns - moved over 1 second ago",
			vulns: []grype.Vulnerability{
				{ID: "CVE-2021-1", Severity: "High"},
			},
			hasMoved:       true,
			timeSinceMoved: 2 * time.Second,
			expectHint:     false,
			expectedText:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &mockVulnerabilitySource{vulns: tt.vulns}
			model := NewModel(source)
			model.windowSizeReceived = true // Mark as received for test
			gameModel := model.startGame(tt.vulns)

			// Set movement state
			gameModel.hasMoved = tt.hasMoved
			if tt.hasMoved {
				gameModel.firstMoveTime = time.Now().Add(-tt.timeSinceMoved)
			}

			// Render the game
			output := gameModel.renderGame()

			// Check if hint is displayed
			if tt.expectHint {
				if !strings.Contains(output, tt.expectedText) {
					t.Errorf("expected hint text '%s' not found in output", tt.expectedText)
				}
			} else {
				// Check that neither hint message appears
				if strings.Contains(output, "Ahhh, so peaceful!") {
					t.Error("unexpected zero-vuln hint found in output")
				}
				if strings.Contains(output, "Make it to the finish line without getting hit") {
					t.Error("unexpected normal hint found in output")
				}
			}
		})
	}
}

func TestDecorativeItems(t *testing.T) {
	source := &mockVulnerabilitySource{vulns: []grype.Vulnerability{}}
	model := NewModel(source)
	model.windowSizeReceived = true // Mark as received for test
	gameModel := model.startGame([]grype.Vulnerability{})

	// Check that decorative items were created for zero-vuln game
	if !gameModel.isZeroVulnGame {
		t.Error("expected isZeroVulnGame to be true")
	}

	if len(gameModel.decorativeItems) == 0 {
		t.Error("expected decorative items to be created")
	}

	// Check that decorative items have expected symbols
	hasHearts := false
	hasStars := false
	for _, item := range gameModel.decorativeItems {
		if item.symbol == "💚" {
			hasHearts = true
		}
		if item.symbol == "✨" || item.symbol == "⭐" {
			hasStars = true
		}
	}

	if !hasHearts {
		t.Error("expected to find heart decorative items")
	}
	if !hasStars {
		t.Error("expected to find star decorative items")
	}

	// Render and check for decorative items in output
	output := gameModel.renderGame()
	if !strings.Contains(output, "💚") {
		t.Error("expected to see hearts in rendered output")
	}
	if !strings.Contains(output, "✨") && !strings.Contains(output, "⭐") {
		t.Error("expected to see stars in rendered output")
	}
}
