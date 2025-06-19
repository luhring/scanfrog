package game

import (
	"testing"

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
	if model.lives != 3 {
		t.Errorf("expected 3 lives, got %d", model.lives)
	}
	if model.vulnSource == nil {
		t.Error("expected vulnSource to be set")
	}
}

func TestWaveSystem(t *testing.T) {
	// Create 150 vulnerabilities to test wave splitting
	vulns := make([]grype.Vulnerability, 150)
	for i := range vulns {
		vulns[i] = grype.Vulnerability{
			ID:       "CVE-2021-" + string(rune(i)),
			Severity: "Medium",
		}
	}
	source := &mockVulnerabilitySource{vulns: vulns}

	model := NewModel(source)
	// Simulate vulnerabilities loaded
	*model = model.startGame(vulns)

	// Should create 3 waves (50 + 50 + 50)
	expectedWaves := 3
	if model.totalWaves != expectedWaves {
		t.Errorf("expected %d waves, got %d", expectedWaves, model.totalWaves)
	}

	// Current wave should start at 0
	if model.currentWave != 0 {
		t.Errorf("expected current wave 0, got %d", model.currentWave)
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
		name     string
		obstacle obstacle
		want     string
	}{
		{
			name: "with CVSS score",
			obstacle: obstacle{
				cveID:         "CVE-2021-12345",
				severity:      8.5,
				severityLabel: "High",
			},
			want: "You were hit by CVE-2021-12345 (High, CVSS 8.5). Game over!",
		},
		{
			name: "without CVSS score",
			obstacle: obstacle{
				cveID:         "CVE-2021-12345",
				severity:      0,
				severityLabel: "Medium",
			},
			want: "You were hit by CVE-2021-12345 (Medium). Game over!",
		},
		{
			name: "CVSS but no label",
			obstacle: obstacle{
				cveID:         "CVE-2021-12345",
				severity:      9.5,
				severityLabel: "",
			},
			want: "You were hit by CVE-2021-12345 (Critical, CVSS 9.5). Game over!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCollisionMessage(tt.obstacle)
			if got != tt.want {
				t.Errorf("formatCollisionMessage() = %v, want %v", got, tt.want)
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
