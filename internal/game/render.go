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

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#1976D2", // Dark blue for light terminals
			Dark:  "#64B5F6", // Light blue for dark terminals
		}).
		Bold(true).
		Align(lipgloss.Center, lipgloss.Center)

	gameOverStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#B71C1C", // Dark red for light terminals
			Dark:  "#F44336", // Bright red for dark terminals
		}).
		Bold(true).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.DoubleBorder()).
		Padding(2, 4)

	victoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#2E7D32", // Dark green for light terminals
			Dark:  "#4CAF50", // Bright green for dark terminals
		}).
		Bold(true).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.DoubleBorder()).
		Padding(2, 4)
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

func (m Model) renderGame() string {
	// Account for header (2 lines)
	gameHeight := m.height - 2
	if gameHeight < 10 {
		gameHeight = 10
	}
	
	// Create game board
	board := make([][]rune, gameHeight)
	for i := range board {
		board[i] = make([]rune, m.width)
		for j := range board[i] {
			board[i][j] = ' '
		}
	}

	// Draw finish line at the top row (y=0)
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

	// Draw lanes
	for _, lane := range m.lanes {
		if lane.y < len(board) {
			for x := 0; x < m.width; x++ {
				if x%4 == 0 || x%4 == 1 {
					board[lane.y][x] = '─'
				}
			}
		}
	}

	// Draw obstacles
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

	// Draw frog (we'll handle emoji rendering in the display loop)
	if m.frog.y < len(board) && m.frog.x < m.width {
		board[m.frog.y][m.frog.x] = 'F' // Placeholder, will be replaced with emoji
	}

	// Convert board to string with styling
	var output strings.Builder

	// Header
	header := fmt.Sprintf("SCANFROG - Wave %d/%d - Lives: %d",
		m.currentWave+1, m.totalWaves, m.lives)
	output.WriteString(scoreStyle.Render(header))
	output.WriteString("\n")
	separator := strings.Repeat("═", m.width)
	output.WriteString(scoreStyle.Render(separator))
	output.WriteString("\n")

	// Finish line at the top - draw it as part of the first row
	// We'll incorporate it into the board rendering instead

	// Game board
	for y, row := range board {
		for x := 0; x < len(row); x++ {
			cell := row[x]

			// Check if this position has an obstacle
			var isObstacle bool
			var obsSeverity float64
			for _, obs := range m.obstacles {
				if obs.pos.y == y && x >= obs.pos.x && x < obs.pos.x+obs.width {
					isObstacle = true
					obsSeverity = obs.severity
					break
				}
			}

			// Check if we need to render an emoji at this position
			skipNext := false

			// Apply styling
			cellStr := string(cell)
			if cell == 'F' && m.frog.y == y && m.frog.x == x {
				// Only render frog emoji at the actual frog position
				cellStr = frogStyle.Render("🐸")
				skipNext = true
			} else if isObstacle {
				// Always show emoji for obstacles, not the CVE text
				if obsSeverity >= 9.0 {
					cellStr = bossStyle.Render("🦖") // T-Rex for critical
					skipNext = true
				} else if obsSeverity >= 7.0 {
					cellStr = truckStyle.Render("🚛") // Truck for high
					skipNext = true
				} else {
					cellStr = carStyle.Render("🚗") // Car for medium/low
					skipNext = true
				}
			} else if cell == '─' {
				cellStr = roadStyle.Render("━")
			} else if y == 0 {
				// Apply finish line styling to the entire top row
				cellStr = finishLineStyle.Render(string(cell))
			}

			output.WriteString(cellStr)

			// If we rendered an emoji, skip the next cell to account for double width
			if skipNext && x < len(row)-1 {
				x++
			}
		}
		output.WriteString("\n")
	}

	return output.String()
}

func (m Model) renderGameOver() string {
	content := fmt.Sprintf("GAME OVER\n\n%s\n\nPress ESC or Q to quit", m.collisionMsg)
	// Account for border (2) and padding (4 horizontal, 4 vertical) in total dimensions
	// The Width/Height in lipgloss is the total box size including border
	boxWidth := m.width - 2   // Leave some margin
	boxHeight := m.height - 2  // Leave some margin
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxHeight < 10 {
		boxHeight = 10
	}
	return gameOverStyle.Width(boxWidth).Height(boxHeight).Render(content)
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
	boxHeight := m.height - 2  // Leave some margin
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxHeight < 15 {
		boxHeight = 15 // Victory screen needs more height
	}
	return victoryStyle.Width(boxWidth).Height(boxHeight).Render(content.String())
}

