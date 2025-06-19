package game

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luhring/scanfrog/internal/grype"
)

type gameState int

const (
	stateLoading gameState = iota
	statePlaying
	stateGameOver
	stateVictory
)

const (
	// gameAreaHeight is the fixed height of the playable game area
	gameAreaHeight = 20
	// minTerminalHeight is the minimum terminal height required
	minTerminalHeight = 22 // gameAreaHeight + 2 for header
)

type position struct {
	x, y int
}

type obstacle struct {
	pos           position
	floatX        float64 // Track precise position
	width         int
	speed         float64
	cveID         string
	severity      float64
	severityLabel string
}

// Model represents the main game state and handles all game logic for the Scanfrog terminal game.
type Model struct {
	vulnSource grype.VulnerabilitySource
	state      gameState

	// Loading state
	loadingMsg string

	// Game state
	frog      position
	obstacles []obstacle
	lanes     []lane

	// Victory tracking
	gameStartTime  time.Time
	totalVulns     int
	containerImage string

	// Game over state
	collisionCVE string
	collisionMsg string
	collisionObs *obstacle // Store the obstacle for rendering

	// Viewport
	width, height int

	// Timing
	lastUpdate time.Time

	// Hint display
	hasMoved        bool
	firstMoveTime   time.Time
	isZeroVulnGame  bool
	decorativeItems []decorativeItem

	// Cached vulnerability data
	loadedVulns []grype.Vulnerability
}

type decorativeItem struct {
	x, y   int
	symbol string
	floatX float64
	floatY float64
	speed  float64
}

type lane struct {
	y         int
	direction int // -1 for left, 1 for right
	speed     float64
}

// NewModel creates a new game model with the specified vulnerability source.
func NewModel(vulnSource grype.VulnerabilitySource) *Model {
	loadingMsg := "Building obstacle course..."
	containerImage := ""
	if scanner, ok := vulnSource.(*grype.ScannerSource); ok {
		containerImage = scanner.Image
		loadingMsg = fmt.Sprintf("Building obstacle course from %s...", scanner.Image)
	}

	return &Model{
		vulnSource:     vulnSource,
		state:          stateLoading,
		loadingMsg:     loadingMsg,
		containerImage: containerImage,
		width:          80,
		height:         24,
		lastUpdate:     time.Now(),
	}
}

// Init initializes the game model and returns commands to load vulnerabilities and set up the terminal.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadVulnerabilities(),
		tea.EnterAltScreen,
		m.tick(), // Start ticking immediately for spinner animation
	)
}

// Update processes incoming messages and updates the game state accordingly.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		// Only update if we get valid dimensions
		if msg.Width > 0 && msg.Height > 0 {
			m.width = msg.Width
			m.height = msg.Height
		}
		return m, nil

	case vulnerabilitiesLoadedMsg:
		m.loadedVulns = msg.vulns
		return m.startGame(msg.vulns), m.tick()

	case vulnerabilityErrorMsg:
		m.state = stateGameOver
		m.collisionMsg = fmt.Sprintf("Failed to load vulnerabilities: %v", msg.err)
		return m, tea.Quit

	case tickMsg:
		switch m.state {
		case statePlaying:
			return m.updateGame(), m.tick()
		case stateLoading:
			// Keep ticking during loading to animate spinner
			return m, m.tick()
		}
		return m, nil
	}

	return m, nil
}

// View renders the current game state as a string for display.
func (m Model) View() string {
	switch m.state {
	case stateLoading:
		return m.renderLoading()
	case statePlaying:
		return m.renderGame()
	case stateGameOver:
		return m.renderGameOver()
	case stateVictory:
		return m.renderVictory()
	default:
		return "Unknown state"
	}
}

type vulnerabilitiesLoadedMsg struct {
	vulns []grype.Vulnerability
}

type vulnerabilityErrorMsg struct {
	err error
}

type tickMsg time.Time

func (m Model) loadVulnerabilities() tea.Cmd {
	return func() tea.Msg {
		vulns, err := m.vulnSource.GetVulnerabilities()
		if err != nil {
			return vulnerabilityErrorMsg{err: err}
		}
		return vulnerabilitiesLoadedMsg{vulns: vulns}
	}
}

func (m Model) tick() tea.Cmd {
	// 30 FPS provides smooth gameplay while reducing CPU usage
	// Physics use delta time, so game speed remains consistent
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) restartGame() (tea.Model, tea.Cmd) {
	// Reset game state while keeping loaded vulnerabilities
	m.state = statePlaying
	m.hasMoved = false
	m.collisionCVE = ""
	m.collisionMsg = ""
	m.collisionObs = nil
	m.decorativeItems = nil
	m.isZeroVulnGame = false
	m.lastUpdate = time.Now()

	// Restart with cached vulnerabilities
	return m.startGame(m.loadedVulns), m.tick()
}
