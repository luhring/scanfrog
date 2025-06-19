package game

import (
	"fmt"
	"math"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luhring/scanfrog/internal/grype"
)

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state != statePlaying {
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "enter", " ":
			if m.state == stateGameOver {
				return m, tea.Quit
			} else if m.state == stateVictory {
				// Restart the game
				newModel := NewModel(m.vulnSource)
				// Preserve the current window dimensions
				newModel.width = m.width
				newModel.height = m.height
				return newModel, newModel.Init()
			}
		}
		return m, nil
	}

	// Game controls
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit

	// Movement
	case "up", "w":
		if m.frog.y > 0 {
			m.frog.y--
		}
	case "down", "s":
		// Account for header (2 lines)
		gameHeight := m.height - 2
		if m.frog.y < gameHeight-1 {
			m.frog.y++
		}
	case "left", "a":
		if m.frog.x > 0 {
			m.frog.x--
		}
	case "right", "d":
		if m.frog.x < m.width-1 {
			m.frog.x++
		}
	}

	// Check win condition
	if m.frog.y == 0 {
		m.currentWave++
		if m.currentWave >= m.totalWaves {
			m.state = stateVictory
			return m, nil // Don't quit, show victory screen
		}
		// Reset frog for next wave (account for header)
		gameHeight := m.height - 2
		m.frog = position{x: m.width / 2, y: gameHeight - 1}
		m.waveTimer = time.Now()
	}

	return m, nil
}

func (m Model) startGame(vulns []grype.Vulnerability) Model {
	m.state = statePlaying
	m.gameStartTime = time.Now()
	m.totalVulns = len(vulns)

	// Calculate waves
	const vulnsPerWave = 50
	m.totalWaves = int(math.Ceil(float64(len(vulns)) / float64(vulnsPerWave)))
	if m.totalWaves == 0 {
		m.totalWaves = 1
	}
	m.currentWave = 0

	// Position frog at bottom center (account for header)
	gameHeight := m.height - 2
	m.frog = position{
		x: m.width / 2,
		y: gameHeight - 1,
	}

	// Initialize lanes - start much closer to the frog
	m.lanes = make([]lane, 0, 10)
	startLane := gameHeight - 3 // Start lanes just above the frog
	for i := 0; i < 10 && startLane-i > 0; i++ {
		m.lanes = append(m.lanes, lane{
			y:         startLane - i,
			direction: 1 - 2*(i%2), // Alternate directions
			speed:     0.5 + float64(i%3)*0.3,
		})
	}

	// Generate initial obstacles
	m.generateObstacles(vulns)

	return m
}

func (m *Model) generateObstacles(vulns []grype.Vulnerability) {
	m.obstacles = nil

	// For this wave, take the appropriate slice of vulnerabilities
	startIdx := m.currentWave * 50
	endIdx := startIdx + 50
	if endIdx > len(vulns) {
		endIdx = len(vulns)
	}

	waveVulns := vulns[startIdx:endIdx]

	// Distribute vulnerabilities across lanes
	for i, vuln := range waveVulns {
		laneIdx := i % len(m.lanes)
		lane := m.lanes[laneIdx]

		// Determine obstacle properties based on severity
		width := 1
		speedMultiplier := 1.0

		if vuln.CVSS >= 9.0 {
			width = 2 // Boss/alligator
			speedMultiplier = 1.5
		} else if vuln.CVSS >= 7.0 {
			width = 2 // Truck
			speedMultiplier = 1.2
		} else if vuln.CVSS >= 4.0 {
			speedMultiplier = 1.3 // Faster car
		}

		// Space obstacles out more evenly across the screen
		xOffset := (i / len(m.lanes)) * 20
		// Add some randomness to initial positions
		startX := xOffset % m.width
		if lane.direction < 0 {
			startX = m.width - (xOffset % m.width)
		}

		m.obstacles = append(m.obstacles, obstacle{
			pos: position{
				x: startX,
				y: lane.y,
			},
			floatX:        float64(startX),
			width:         width,
			speed:         lane.speed * speedMultiplier * float64(lane.direction),
			cveID:         vuln.ID,
			severity:      vuln.CVSS,
			severityLabel: vuln.Severity,
		})
	}
}

func (m Model) updateGame() Model {
	now := time.Now()
	delta := now.Sub(m.lastUpdate).Seconds()
	m.lastUpdate = now

	// Update obstacle positions with floating point precision
	for i := range m.obstacles {
		// Move obstacles based on their speed and delta time
		movement := m.obstacles[i].speed * delta * 30.0
		m.obstacles[i].floatX += movement
		m.obstacles[i].pos.x = int(m.obstacles[i].floatX)

		// Wrap around screen
		if m.obstacles[i].pos.x < -m.obstacles[i].width-5 {
			m.obstacles[i].floatX = float64(m.width + 5)
			m.obstacles[i].pos.x = m.width + 5
		} else if m.obstacles[i].pos.x > m.width+5 {
			m.obstacles[i].floatX = float64(-m.obstacles[i].width - 5)
			m.obstacles[i].pos.x = -m.obstacles[i].width - 5
		}
	}

	// Check collisions
	for _, obs := range m.obstacles {
		if m.checkCollision(m.frog, obs) {
			m.state = stateGameOver
			m.collisionCVE = obs.cveID
			m.collisionMsg = formatCollisionMessage(obs)
			return m
		}
	}

	return m
}

func (m Model) checkCollision(frog position, obs obstacle) bool {
	if frog.y != obs.pos.y {
		return false
	}

	// Check if frog x position overlaps with obstacle
	return frog.x >= obs.pos.x && frog.x < obs.pos.x+obs.width
}

func formatCollisionMessage(obs obstacle) string {
	// Use the actual severity label from Grype
	severity := obs.severityLabel
	if severity == "" {
		// Fallback to CVSS-based severity if label is missing
		if obs.severity >= 9.0 {
			severity = "Critical"
		} else if obs.severity >= 7.0 {
			severity = "High"
		} else if obs.severity >= 4.0 {
			severity = "Medium"
		} else {
			severity = "Low"
		}
	}

	return fmt.Sprintf("You were hit by %s (%s, CVSS %.1f). Game over!",
		obs.cveID, severity, obs.severity)
}

