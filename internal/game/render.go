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
	idx := int(time.Now().UnixMilli()/100) % len(spinner)
	
	content := fmt.Sprintf("%s %s", spinner[idx], m.loadingMsg)
	return loadingStyle.Width(m.width).Height(m.height).Render(content)
}

func (m Model) renderGame() string {
	// Create game board
	board := make([][]rune, m.height)
	for i := range board {
		board[i] = make([]rune, m.width)
		for j := range board[i] {
			board[i][j] = ' '
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
	
	// Draw frog
	if m.frog.y < len(board) && m.frog.x < m.width {
		board[m.frog.y][m.frog.x] = '🐸'
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
	
	// Game board
	for y, row := range board {
		for x, cell := range row {
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
			
			// Apply styling
			cellStr := string(cell)
			if cell == '🐸' {
				cellStr = frogStyle.Render("🐸")
			} else if isObstacle {
				// Use better visual representations
				if obsSeverity >= 9.0 {
					if cell == '█' || cell == ' ' {
						cellStr = bossStyle.Render("🦖") // T-Rex for critical
					} else {
						cellStr = bossStyle.Render(cellStr)
					}
				} else if obsSeverity >= 7.0 {
					if cell == '█' || cell == ' ' {
						cellStr = truckStyle.Render("🚛") // Truck for high
					} else {
						cellStr = truckStyle.Render(cellStr)
					}
				} else {
					if cell == '█' || cell == ' ' {
						cellStr = carStyle.Render("🚗") // Car for medium/low
					} else {
						cellStr = carStyle.Render(cellStr)
					}
				}
			} else if cell == '─' {
				cellStr = roadStyle.Render("━")
			}
			
			output.WriteString(cellStr)
		}
		output.WriteString("\n")
	}
	
	return output.String()
}

func (m Model) renderGameOver() string {
	content := fmt.Sprintf("GAME OVER\n\n%s\n\nPress ESC or Q to quit", m.collisionMsg)
	return gameOverStyle.Render(content)
}

func (m Model) renderVictory() string {
	content := "🎉 VICTORY! 🎉\n\nYou successfully navigated all vulnerabilities!\n\nYour container is secure!\n\nPress any key to exit"
	return victoryStyle.Render(content)
}