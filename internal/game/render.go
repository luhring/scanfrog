package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	frogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#2E7D32", // Dark green for light terminals
			Dark:  "#4CAF50", // Bright green for dark terminals
		}).
		Bold(true)

	finishLineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#FFFFFF", // White for light terminals
			Dark:  "#FFFF00", // Yellow for dark terminals
		}).
		Background(lipgloss.AdaptiveColor{
			Light: "#000000", // Black background for light terminals
			Dark:  "#1976D2", // Blue background for dark terminals
		}).
		Bold(true)

	carStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#E65100", // Dark orange for light terminals
			Dark:  "#FF9800", // Bright orange for dark terminals
		})

	truckStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#B71C1C", // Dark red for light terminals
			Dark:  "#F44336", // Bright red for dark terminals
		}).
		Bold(true)

	bossStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#6A1B9A", // Dark purple for light terminals
			Dark:  "#E91E63", // Bright pink for dark terminals
		}).
		Bold(true).
		Blink(true)

	roadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#757575", // Medium gray for light terminals
			Dark:  "#616161", // Dark gray for dark terminals
		})

	scoreStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#F57C00", // Dark amber for light terminals
			Dark:  "#FFC107", // Bright amber for dark terminals
		}).
		Bold(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#BDBDBD", // Light gray for light terminals
			Dark:  "#424242", // Dark gray for dark terminals
		})

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#1976D2", // Dark blue for light terminals
			Dark:  "#64B5F6", // Light blue for dark terminals
		}).
		Bold(true).
		Align(lipgloss.Center, lipgloss.Center)

	victoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#2E7D32", // Dark green for light terminals
			Dark:  "#4CAF50", // Bright green for dark terminals
		}).
		Bold(true).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.DoubleBorder()).
		Padding(2, 4)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#757575", // Medium gray for light terminals
			Dark:  "#9E9E9E", // Light gray for dark terminals
		}).
		Faint(true).
		Align(lipgloss.Center)

	decorativeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#4CAF50", // Green for light terminals
			Dark:  "#81C784", // Light green for dark terminals
		})

	// Add a style for bicycles (negligible)
	bicycleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#1976D2", // Blue for light terminals
			Dark:  "#64B5F6", // Light blue for dark terminals
		})

	// Add a style for low severity cars
	lowCarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#388E3C", // Green for light terminals
			Dark:  "#81C784", // Light green for dark terminals
		})

	// Add a style for medium severity cars
	mediumCarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#FBC02D", // Yellow for light terminals
			Dark:  "#FFF176", // Light yellow for dark terminals
		})
)

func (m Model) renderLoading() string {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	idx := int(time.Now().UnixMilli()/50) % len(spinner) // Twice as fast

	content := fmt.Sprintf("%s %s", spinner[idx], m.loadingMsg)
	// Ensure we don't exceed terminal bounds
	width := m.width
	height := m.height
	if width < 1 {
		width = 80 // Default width
	}
	if height < 1 {
		height = 24 // Default height
	}
	return loadingStyle.Width(width).Height(height).Render(content)
}

// initializeBoard creates an empty game board
func (m Model) initializeBoard() [][]rune {
	board := make([][]rune, gameAreaHeight)
	for i := range board {
		board[i] = make([]rune, m.width)
		for j := range board[i] {
			board[i][j] = ' '
		}
	}
	return board
}

// drawFinishLine draws the finish line at the top of the board
func (m Model) drawFinishLine(board [][]rune) {
	// Use checkered pattern for better visibility
	for x := 0; x < m.width; x++ {
		if x%2 == 0 {
			board[0][x] = '█'
		} else {
			board[0][x] = ' '
		}
	}
	// Add "FINISH" text in the center with padding
	finishText := "  FINISH  "
	finishStart := (m.width - len(finishText)) / 2
	if finishStart >= 0 && finishStart+len(finishText) <= m.width {
		for i, ch := range finishText {
			if finishStart+i < m.width {
				board[0][finishStart+i] = ch
			}
		}
	}
}

// drawLanes draws the road lanes on the board
func (m Model) drawLanes(board [][]rune) {
	for _, lane := range m.lanes {
		if lane.y < len(board) {
			for x := 0; x < m.width; x++ {
				if x%4 == 0 || x%4 == 1 {
					board[lane.y][x] = '─'
				}
			}
		}
	}
}

// drawObstacles draws the obstacles on the board
func (m Model) drawObstacles(board [][]rune) {
	for _, obs := range m.obstacles {
		if obs.pos.y < len(board) && obs.pos.x >= 0 && obs.pos.x < m.width {
			// Truncate CVE ID to fit
			label := obs.cveID
			if len(label) > 10 {
				label = label[:10]
			}

			for i := 0; i < obs.width && obs.pos.x+i < m.width; i++ {
				if obs.pos.x+i >= 0 {
					if i < len(label) {
						board[obs.pos.y][obs.pos.x+i] = rune(label[i])
					} else {
						board[obs.pos.y][obs.pos.x+i] = '█'
					}
				}
			}
		}
	}
}

// drawDecorativeItems draws decorative items for zero-vuln games
func (m Model) drawDecorativeItems(board [][]rune) {
	if m.isZeroVulnGame {
		for _, item := range m.decorativeItems {
			if item.y >= 0 && item.y < len(board) && item.x >= 0 && item.x < m.width {
				board[item.y][item.x] = 'D' // Placeholder for decorative item
			}
		}
	}
}

// calculateTopMargin calculates the top margin for vertical centering
func (m Model) calculateTopMargin() int {
	if m.height > minTerminalHeight {
		return (m.height - minTerminalHeight) / 2
	}
	return 0
}

// renderHeader renders the game header with image name and vulnerability count
func (m Model) renderHeader(output *strings.Builder) {
	headerText := "scanfrog"
	if m.containerImage != "" {
		headerText = fmt.Sprintf("scanfrog • %s", m.containerImage)
		if m.totalVulns > 0 {
			headerText = fmt.Sprintf("scanfrog • %s • %d vulnerabilities", m.containerImage, m.totalVulns)
		}
	}
	output.WriteString(scoreStyle.Render(headerText))
	output.WriteString("\n")
	separator := strings.Repeat("─", m.width)
	output.WriteString(separatorStyle.Render(separator))
	output.WriteString("\n")
}

// renderHintRow renders the special hint row (row 2)
func (m Model) renderHintRow(row []rune, output *strings.Builder) {
	switch {
	case m.frog.y == 2:
		// Normal row rendering for row 2 when frog is present
		for x := 0; x < len(row); x++ {
			cell := row[x]
			cellStr := string(cell)
			if cell == 'F' && m.frog.x == x {
				cellStr = frogStyle.Render("🐸")
				x++ // Skip next cell for emoji width
			}
			output.WriteString(cellStr)
		}
		output.WriteString("\n")
	case !m.hasMoved || time.Since(m.firstMoveTime) < time.Second:
		// Show hint text when frog is not on row 2
		var hintText string
		if m.isZeroVulnGame {
			hintText = "Ahhh, so peaceful! (And boring!) Proceed to the finish line to win!"
		} else {
			hintText = "Make it to the finish line without getting hit by anything!"
		}
		hintStyled := hintStyle.Width(m.width).Render(hintText)
		output.WriteString(hintStyled)
		output.WriteString("\n")
	default:
		// No hint, no frog - just empty row
		output.WriteString("\n")
	}
}

// findObstacleAt finds an obstacle at the given position
func (m Model) findObstacleAt(x, y int) (bool, float64, string) {
	for _, obs := range m.obstacles {
		if obs.pos.y == y && x >= obs.pos.x && x < obs.pos.x+obs.width {
			return true, obs.severity, obs.severityLabel
		}
	}
	return false, 0, ""
}

// findDecorativeItemAt finds a decorative item at the given position
func (m Model) findDecorativeItemAt(x, y int) (bool, string) {
	if m.isZeroVulnGame {
		for _, item := range m.decorativeItems {
			if item.y == y && item.x == x {
				return true, item.symbol
			}
		}
	}
	return false, ""
}

// renderNormalRow renders a normal game board row
func (m Model) renderNormalRow(row []rune, y int, output *strings.Builder) {
	for x := 0; x < len(row); x++ {
		cell := row[x]
		cellStr := m.getCellDisplay(cell, x, y)
		output.WriteString(cellStr)

		// If we rendered an emoji, skip the next cell to account for double width
		if m.shouldSkipNext(cell, x, y) && x < len(row)-1 {
			x++
		}
	}
}

// getCellDisplay returns the styled string for a cell
func (m Model) getCellDisplay(cell rune, x, y int) string {
	// Check if frog is at this position
	if cell == 'F' && m.frog.y == y && m.frog.x == x {
		return frogStyle.Render("🐸")
	}

	// Check for decorative item
	if isDecorativeItem, symbol := m.findDecorativeItemAt(x, y); isDecorativeItem {
		return decorativeStyle.Render(symbol)
	}

	// Check for obstacle
	if isObstacle, severity, severityLabel := m.findObstacleAt(x, y); isObstacle {
		return m.getObstacleEmoji(severity, severityLabel)
	}

	// Apply other styling
	switch {
	case cell == '─':
		return roadStyle.Render("━")
	case y == 0:
		// Apply finish line styling to the entire top row
		return finishLineStyle.Render(string(cell))
	default:
		return string(cell)
	}
}

// shouldSkipNext returns true if the next cell should be skipped (for emoji width)
func (m Model) shouldSkipNext(cell rune, x, y int) bool {
	// Check if frog is at this position
	if cell == 'F' && m.frog.y == y && m.frog.x == x {
		return true
	}

	// Check for decorative item
	if isDecorativeItem, _ := m.findDecorativeItemAt(x, y); isDecorativeItem {
		return true
	}

	// Check for obstacle
	if isObstacle, _, _ := m.findObstacleAt(x, y); isObstacle {
		return true
	}

	return false
}

func (m Model) renderGame() string {
	// Create and populate the game board
	board := m.initializeBoard()
	m.drawFinishLine(board)
	m.drawLanes(board)
	m.drawObstacles(board)
	m.drawDecorativeItems(board)

	// Draw frog (we'll handle emoji rendering in the display loop)
	if m.frog.y < len(board) && m.frog.x < m.width {
		board[m.frog.y][m.frog.x] = 'F' // Placeholder, will be replaced with emoji
	}

	// Convert board to string with styling
	var output strings.Builder

	// Add top margin
	topMargin := m.calculateTopMargin()
	for i := 0; i < topMargin; i++ {
		output.WriteString("\n")
	}

	// Render header
	m.renderHeader(&output)

	// Game board
	for y, row := range board {
		// Special handling for row 2 - hint area
		if y == 2 {
			m.renderHintRow(row, &output)
			continue
		}

		// Normal row rendering
		m.renderNormalRow(row, y, &output)
		// Only add newline if not the last row
		if y < len(board)-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}

func (m Model) renderGameOver() string {
	// Create a style without borders for the content
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{
			Light: "#B71C1C", // Dark red for light terminals
			Dark:  "#F44336", // Bright red for dark terminals
		}).
		Bold(true).
		Align(lipgloss.Center)

	// Build collision message line with proper styling
	var collisionLine string
	if m.collisionObs != nil {
		parts := FormatCollisionMessageParts(m.collisionObs)
		// Style each part separately to avoid style conflicts with hyperlink
		collisionLine = contentStyle.Render(parts.Prefix) + parts.VulnID + contentStyle.Render(parts.Suffix)
	} else {
		// Fallback to plain message
		collisionLine = contentStyle.Render(m.collisionMsg)
	}

	// Render each line with proper styling
	lines := []string{
		contentStyle.Render("GAME OVER"),
		"",
		collisionLine,
		"",
		contentStyle.Render("Press ENTER to try again"),
		contentStyle.Render("Press Q to quit"),
	}

	content := strings.Join(lines, "\n")

	// Account for border (2) and padding (4 horizontal, 4 vertical) in total dimensions
	// The Width/Height in lipgloss is the total box size including border
	boxWidth := m.width - 2   // Leave some margin
	boxHeight := m.height - 2 // Leave some margin
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxHeight < 10 {
		boxHeight = 10
	}

	// Apply the outer box style (with border) to the pre-styled content
	boxStyle := lipgloss.NewStyle().
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.DoubleBorder()).
		Padding(2, 4).
		Width(boxWidth).
		Height(boxHeight)

	return boxStyle.Render(content)
}

func (m Model) renderVictory() string {
	duration := time.Since(m.gameStartTime)
	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds()) % 60

	var content strings.Builder
	content.WriteString("🎉 VICTORY! 🎉\n\n")

	if m.containerImage != "" {
		content.WriteString(fmt.Sprintf("You survived %s!\n\n", m.containerImage))
	} else {
		content.WriteString("You survived the vulnerability gauntlet!\n\n")
	}

	content.WriteString("Statistics:\n")
	content.WriteString(fmt.Sprintf("• Vulnerabilities dodged: %d\n", m.totalVulns))
	content.WriteString(fmt.Sprintf("• Time taken: %dm %ds\n", minutes, seconds))
	content.WriteString("\nThe container lives to deploy another day!\n\n")
	content.WriteString("Press ENTER to play again\n")
	content.WriteString("Press Q to quit")

	// Account for border (2) and padding (4 horizontal, 4 vertical) in total dimensions
	// The Width/Height in lipgloss is the total box size including border
	boxWidth := m.width - 2   // Leave some margin
	boxHeight := m.height - 2 // Leave some margin
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxHeight < 15 {
		boxHeight = 15 // Victory screen needs more height
	}
	return victoryStyle.Width(boxWidth).Height(boxHeight).Render(content.String())
}

// getObstacleEmoji returns the appropriate emoji for an obstacle based on its severity
func (m Model) getObstacleEmoji(cvssScore float64, severityLabel string) string {
	// First check CVSS score if available
	if cvssScore > 0 {
		switch {
		case cvssScore >= 9.0:
			return bossStyle.Render("🦖") // T-Rex for critical
		case cvssScore >= 7.0:
			return truckStyle.Render("🚛") // Truck for high
		case cvssScore >= 4.0:
			return mediumCarStyle.Render("🚗") // Car for medium
		case cvssScore > 0:
			return lowCarStyle.Render("🚗") // Car for low
		default:
			return bicycleStyle.Render("🚲") // Bicycle for negligible
		}
	}

	// Fall back to severity label when no CVSS
	switch severityLabel {
	case "Critical":
		return bossStyle.Render("🦖")
	case "High":
		return truckStyle.Render("🚛")
	case "Medium":
		return mediumCarStyle.Render("🚗")
	case "Low":
		return lowCarStyle.Render("🚗")
	case "Negligible":
		return bicycleStyle.Render("🚲")
	default:
		return carStyle.Render("🚗")
	}
}
