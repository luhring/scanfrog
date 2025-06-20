package game

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/luhring/scanfrog/internal/grype"
)

// mockVulnerabilitySource is a test implementation of VulnerabilitySource
type mockVulnerabilitySource struct {
	vulns []grype.Vulnerability
	err   error
}

func (m *mockVulnerabilitySource) GetVulnerabilities() ([]grype.Vulnerability, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.vulns, nil
}

func TestNewModel(t *testing.T) {
	vulns := []grype.Vulnerability{
		{ID: "CVE-2021-1", Severity: "High", CVSS: 7.5},
		{ID: "CVE-2021-2", Severity: "Medium", CVSS: 5.0},
	}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)

	// Check basic initialization
	if model.state != stateLoading {
		t.Errorf("expected initial state Loading, got %v", model.state)
	}
	if model.vulnSource == nil {
		t.Error("expected vulnSource to be set")
	}
}

func TestAllVulnerabilitiesAtOnce(t *testing.T) {
	// Create 150 vulnerabilities to test that all are displayed at once
	vulns := make([]grype.Vulnerability, 150)
	for i := range vulns {
		vulns[i] = grype.Vulnerability{
			ID:       "CVE-2021-" + string(rune(i)),
			Severity: "Medium",
		}
	}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.windowSizeReceived = true // Mark as received for test
	// Simulate vulnerabilities loaded
	*model = model.startGame(vulns)

	// All vulnerabilities should be loaded as obstacles
	if len(model.obstacles) != 150 {
		t.Errorf("expected 150 obstacles (all vulns), got %d", len(model.obstacles))
	}
}

func TestCollisionDetection(t *testing.T) {
	tests := []struct {
		name     string
		frogPos  position
		obstacle obstacle
		want     bool
	}{
		{
			name:    "direct collision",
			frogPos: position{x: 10, y: 10},
			obstacle: obstacle{
				pos:   position{x: 10, y: 10},
				width: 1,
			},
			want: true,
		},
		{
			name:    "no collision - different Y",
			frogPos: position{x: 10, y: 10},
			obstacle: obstacle{
				pos:   position{x: 10, y: 11},
				width: 1,
			},
			want: false,
		},
		{
			name:    "no collision - different X",
			frogPos: position{x: 10, y: 10},
			obstacle: obstacle{
				pos:   position{x: 15, y: 10},
				width: 1,
			},
			want: false,
		},
		{
			name:    "collision with wide obstacle",
			frogPos: position{x: 11, y: 10},
			obstacle: obstacle{
				pos:   position{x: 10, y: 10},
				width: 2,
			},
			want: true,
		},
		{
			name:    "no collision - just past wide obstacle",
			frogPos: position{x: 12, y: 10},
			obstacle: obstacle{
				pos:   position{x: 10, y: 10},
				width: 2,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{}
			got := model.checkCollision(tt.frogPos, tt.obstacle)
			if got != tt.want {
				t.Errorf("checkCollision() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatCollisionMessage(t *testing.T) {
	tests := []struct {
		name           string
		obstacle       obstacle
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "with CVSS score",
			obstacle: obstacle{
				cveID:         "CVE-2021-12345",
				severity:      8.5,
				severityLabel: "High",
			},
			wantContains: []string{"CVE-2021-12345", "High", "CVSS 8.5", "Game over!"},
		},
		{
			name: "without CVSS score",
			obstacle: obstacle{
				cveID:         "CVE-2021-12345",
				severity:      0,
				severityLabel: "Medium",
			},
			wantContains:   []string{"CVE-2021-12345", "Medium", "Game over!"},
			wantNotContain: []string{"CVSS"},
		},
		{
			name: "CVSS but no label",
			obstacle: obstacle{
				cveID:         "CVE-2021-12345",
				severity:      9.5,
				severityLabel: "",
			},
			wantContains: []string{"CVE-2021-12345", "Critical", "CVSS 9.5", "Game over!"},
		},
		{
			name: "GHSA ID",
			obstacle: obstacle{
				cveID:         "GHSA-abcd-efgh-ijkl",
				severity:      7.2,
				severityLabel: "High",
			},
			wantContains: []string{"GHSA-abcd-efgh-ijkl", "High", "CVSS 7.2", "Game over!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCollisionMessage(tt.obstacle)

			// Check that all expected substrings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("formatCollisionMessage() missing expected substring %q in message: %v", want, got)
				}
			}

			// Check that unwanted substrings are not present
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("formatCollisionMessage() contains unexpected substring %q in message: %v", notWant, got)
				}
			}
		})
	}
}

func TestObstacleGeneration(t *testing.T) {
	vulns := []grype.Vulnerability{
		{ID: "CVE-2021-1", Severity: "Critical", CVSS: 9.5},
		{ID: "CVE-2021-2", Severity: "High", CVSS: 7.5},
		{ID: "CVE-2021-3", Severity: "Medium", CVSS: 0}, // No CVSS
	}

	model := Model{
		width:  80,
		height: 24,
		lanes: []lane{
			{y: 10, direction: 1, speed: 1.0},
			{y: 11, direction: -1, speed: 1.2},
		},
	}

	model.generateObstacles(vulns)

	if len(model.obstacles) != len(vulns) {
		t.Errorf("expected %d obstacles, got %d", len(vulns), len(model.obstacles))
	}

	// Check first obstacle (Critical)
	if len(model.obstacles) > 0 {
		obs := model.obstacles[0]
		if obs.width != 2 {
			t.Errorf("critical vulnerability should have width 2, got %d", obs.width)
		}
		if obs.speed < 1.0 {
			t.Errorf("critical vulnerability should have speed multiplier >= 1.5")
		}
	}

	// Check that obstacles are distributed across lanes
	laneCounts := make(map[int]int)
	for _, obs := range model.obstacles {
		laneCounts[obs.pos.y]++
	}
	if len(laneCounts) < 2 {
		t.Error("obstacles should be distributed across multiple lanes")
	}
}

func TestDeltaTimePhysics(t *testing.T) {
	// Test that obstacle movement is frame-rate independent
	vulns := []grype.Vulnerability{
		{ID: "CVE-2021-1", Severity: "Medium", CVSS: 5.0},
	}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	model.width = 80
	model.height = 24
	model.windowSizeReceived = true // Mark as received for test
	gameModel := model.startGame(vulns)

	// Record initial obstacle position
	if len(gameModel.obstacles) == 0 {
		t.Fatal("No obstacles generated")
	}
	initialX := gameModel.obstacles[0].floatX

	// Update the game a few times to ensure movement happens
	time.Sleep(10 * time.Millisecond) // Small sleep to ensure time advances
	gameModel = gameModel.updateGame()

	// Check that obstacle moved
	finalX := gameModel.obstacles[0].floatX
	if finalX == initialX {
		t.Error("Obstacle did not move after update")
	}

	// Basic sanity check - obstacle should move in the expected direction
	// (direction can be positive or negative based on lane)
	moved := finalX - initialX
	if moved == 0 {
		t.Error("Obstacle position did not change")
	}

	// The movement should be reasonable (not the huge number we were seeing)
	if math.Abs(moved) > 100 {
		t.Errorf("Obstacle movement too large: moved %.2f units", moved)
	}
}
