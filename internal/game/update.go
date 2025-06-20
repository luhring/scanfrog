package game

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luhring/scanfrog/internal/grype"
	"github.com/savioxavier/termlink"
)

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state != statePlaying {
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "enter", " ":
			if m.state == stateGameOver || m.state == stateVictory {
				// Restart the game using cached vulnerabilities
				return m.restartGame()
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
			if !m.hasMoved {
				m.hasMoved = true
				m.firstMoveTime = time.Now()
			}
		}
	case "down", "s":
		if m.frog.y < gameAreaHeight-1 {
			m.frog.y++
			if !m.hasMoved {
				m.hasMoved = true
				m.firstMoveTime = time.Now()
			}
		}
	case "left", "a":
		if m.frog.x > 0 {
			m.frog.x--
			if !m.hasMoved {
				m.hasMoved = true
				m.firstMoveTime = time.Now()
			}
		}
	case "right", "d":
		if m.frog.x < m.width-1 {
			m.frog.x++
			if !m.hasMoved {
				m.hasMoved = true
				m.firstMoveTime = time.Now()
			}
		}
	}

	// Check win condition
	if m.frog.y == 0 {
		m.state = stateVictory
		return m, nil // Don't quit, show victory screen
	}

	return m, nil
}

func (m Model) startGame(vulns []grype.Vulnerability) Model {
	m.state = statePlaying
	m.gameStartTime = time.Now()
	m.totalVulns = len(vulns)

	// Position frog at bottom of game area
	m.frog = position{
		x: m.width / 2,
		y: gameAreaHeight - 1,
	}

	// Initialize lanes with proper spacing
	// We want lanes at: 18, 16, 14, 12, 10, 8, 6, 4
	// This gives us:
	// - Row 19: frog start position (empty)
	// - Row 18: road lane (bottom)
	// - Row 17: empty
	// - Row 16: road lane
	// - Row 15: empty
	// - Row 14: road lane
	// - ... continuing with alternating pattern
	// - Row 4: road lane (top)
	// - Row 3: empty
	// - Row 2: hint/empty
	// - Row 1: empty
	// - Row 0: finish line
	m.lanes = make([]lane, 0, 8)
	lanePositions := []int{18, 16, 14, 12, 10, 8, 6, 4}
	for i, y := range lanePositions {
		if y < gameAreaHeight {
			m.lanes = append(m.lanes, lane{
				y:         y,
				direction: 1 - 2*(i%2), // Alternate directions
				speed:     0.5 + float64(i%3)*0.3,
			})
		}
	}

	// Generate initial obstacles
	m.generateObstacles(vulns)

	// Check if this is a zero-vulnerability game
	if len(vulns) == 0 {
		m.isZeroVulnGame = true
		m.initializeDecorativeItems()
	}

	// Initialize last update time for delta time calculations
	m.lastUpdate = time.Now()

	return m
}

// obstacleType represents the type of obstacle based on severity
type obstacleType int

const (
	obstacleTypeCar obstacleType = iota
	obstacleTypeTruck
	obstacleTypeBoss
)

// getObstacleProperties determines the properties of an obstacle based on vulnerability severity
func getObstacleProperties(vuln grype.Vulnerability) (width int, speedMultiplier float64, obsType obstacleType) {
	// Default values
	width = 1
	speedMultiplier = 1.0
	obsType = obstacleTypeCar

	// First try CVSS score if available
	if vuln.CVSS > 0 {
		switch {
		case vuln.CVSS >= 9.0:
			width = 2 // Boss/T-Rex
			speedMultiplier = 1.5
			obsType = obstacleTypeBoss
		case vuln.CVSS >= 7.0:
			width = 2 // Truck
			speedMultiplier = 1.2
			obsType = obstacleTypeTruck
		case vuln.CVSS >= 4.0:
			speedMultiplier = 1.3 // Faster car
		}
		return
	}

	// Fall back to severity label when no CVSS
	switch vuln.Severity {
	case "Critical":
		width = 2
		speedMultiplier = 1.5
		obsType = obstacleTypeBoss
	case "High":
		width = 2
		speedMultiplier = 1.2
		obsType = obstacleTypeTruck
	case "Medium":
		speedMultiplier = 1.3
	case "Low":
		speedMultiplier = 1.0
	case "Negligible":
		speedMultiplier = 0.8
	}
	return
}

func (m *Model) generateObstacles(vulns []grype.Vulnerability) {
	m.obstacles = nil

	numLanes := len(m.lanes)
	if numLanes == 0 {
		return
	}

	// Each vulnerability becomes exactly one obstacle
	for i, vuln := range vulns {
		laneIdx := i % numLanes
		lane := m.lanes[laneIdx]

		// Get obstacle properties
		width, speedMultiplier, _ := getObstacleProperties(vuln)

		// For 471 vulnerabilities across 8 lanes, we get ~59 per lane
		// We need to pack them tightly to see many on screen at once
		obstacleIndexInLane := i / numLanes

		// Use very tight spacing for high vulnerability counts
		var spacing float64
		switch {
		case len(vulns) > 200:
			spacing = 8.0 // Minimum comfortable spacing
		case len(vulns) > 100:
			spacing = 12.0
		default:
			spacing = 20.0
		}

		// Position based on index with some randomness
		baseX := float64(obstacleIndexInLane) * spacing

		// Add lane-specific offset to stagger
		laneOffset := float64(laneIdx) * 2.0

		// Small random variation
		variation := float64(i%7-3) * 0.5

		x := baseX + laneOffset + variation

		// CRITICAL: Wrap positions to create a continuous loop
		// This ensures consistent density regardless of screen width
		loopLength := float64(len(vulns)/numLanes) * spacing
		x = math.Mod(x, loopLength)
		if x < 0 {
			x += loopLength
		}

		// Start positions distributed around the loop
		var startX int
		if lane.direction > 0 {
			// Moving right: distribute from left
			startX = int(x) - int(loopLength)/2
		} else {
			// Moving left: distribute from right
			startX = int(x) + m.width - int(loopLength)/2
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

	// Update decorative items for zero-vuln games
	if m.isZeroVulnGame {
		for i := range m.decorativeItems {
			// Gentle horizontal floating
			m.decorativeItems[i].floatX += m.decorativeItems[i].speed * delta * 10.0

			// Add a subtle vertical bobbing effect
			bobAmount := math.Sin(float64(now.UnixMilli())/1000.0+float64(i)) * 0.5
			m.decorativeItems[i].floatY += bobAmount * delta

			// Update integer positions
			m.decorativeItems[i].x = int(m.decorativeItems[i].floatX)
			m.decorativeItems[i].y = int(m.decorativeItems[i].floatY)

			// Wrap around screen horizontally
			if m.decorativeItems[i].x > m.width+2 {
				m.decorativeItems[i].floatX = -2.0
				m.decorativeItems[i].x = -2
			}
		}
	}

	// Check collisions
	for _, obs := range m.obstacles {
		if m.checkCollision(m.frog, obs) {
			m.state = stateGameOver
			m.collisionCVE = obs.cveID
			m.collisionMsg = formatCollisionMessage(obs)
			obsCopy := obs // Make a copy to avoid pointer to loop variable
			m.collisionObs = &obsCopy
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

// getVulnerabilityURL returns the appropriate URL for a vulnerability ID
func getVulnerabilityURL(vulnID string) string {
	if strings.HasPrefix(vulnID, "CVE-") {
		// CVE format: https://nvd.nist.gov/vuln/detail/CVE-2024-10041
		return fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vulnID)
	} else if strings.HasPrefix(vulnID, "GHSA-") {
		// GHSA format: https://github.com/advisories/GHSA-xxxx-xxxx-xxxx
		return fmt.Sprintf("https://github.com/advisories/%s", vulnID)
	}
	// If it's neither CVE nor GHSA, return empty string (no link)
	return ""
}

// CollisionMessageParts contains the parts of the collision message for proper rendering
type CollisionMessageParts struct {
	Prefix string // "You were hit by "
	VulnID string // The vulnerability ID (may contain hyperlink)
	Suffix string // " (High, CVSS 7.5). Game over!"
}

func formatCollisionMessage(obs obstacle) string {
	parts := FormatCollisionMessageParts(obs)
	return parts.Prefix + parts.VulnID + parts.Suffix
}

// FormatCollisionMessageParts splits the collision message into parts for proper rendering
func FormatCollisionMessageParts(obs obstacle) CollisionMessageParts {
	// Use the actual severity label from Grype
	severity := obs.severityLabel
	if severity == "" {
		// Fallback to CVSS-based severity if label is missing
		switch {
		case obs.severity >= 9.0:
			severity = "Critical"
		case obs.severity >= 7.0:
			severity = "High"
		case obs.severity >= 4.0:
			severity = "Medium"
		default:
			severity = "Low"
		}
	}

	parts := CollisionMessageParts{
		Prefix: "You were hit by ",
	}

	// Add the vulnerability ID (with or without hyperlink)
	if url := getVulnerabilityURL(obs.cveID); url != "" {
		// termlink.Link will create a clickable hyperlink in supported terminals
		// and fall back to plain text in unsupported terminals
		parts.VulnID = termlink.Link(obs.cveID, url)
	} else {
		parts.VulnID = obs.cveID
	}

	// Add severity info
	if obs.severity > 0 {
		parts.Suffix = fmt.Sprintf(" (%s, CVSS %.1f). Game over!", severity, obs.severity)
	} else {
		parts.Suffix = fmt.Sprintf(" (%s). Game over!", severity)
	}

	return parts
}

func (m *Model) initializeDecorativeItems() {
	m.decorativeItems = nil

	// Create about 10-15 floating hearts and stars
	symbols := []string{"💚", "✨", "💚", "⭐", "💚", "✨"}

	for i := 0; i < 12; i++ {
		// Distribute across the screen, avoiding the frog's starting position
		x := (i * m.width / 12) + (i % 3) - 1
		y := 1 + (i % (gameAreaHeight - 2)) // Start at row 1, avoid finish line and bottom

		// Don't place on the frog's starting position
		if y == m.frog.y && x >= m.frog.x-2 && x <= m.frog.x+2 {
			y = (y + 3) % (gameAreaHeight - 1)
			if y == 0 {
				y = 1 // Avoid finish line
			}
		}

		m.decorativeItems = append(m.decorativeItems, decorativeItem{
			x:      x,
			y:      y,
			symbol: symbols[i%len(symbols)],
			floatX: float64(x),
			floatY: float64(y),
			speed:  0.3 + float64(i%3)*0.2, // Gentle floating speed
		})
	}
}
