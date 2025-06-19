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

type Model struct {
	vulnSource grype.VulnerabilitySource
	state      gameState

	// Loading state
	loadingMsg string

	// Game state
	frog      position
	obstacles []obstacle
	lanes     []lane
	score     int
	lives     int

	// Wave management
	currentWave  int
	totalWaves   int
	waveTimer    time.Time
	waveDuration time.Duration

	// Victory tracking
	gameStartTime  time.Time
	totalVulns     int
	containerImage string

	// Game over state
	collisionCVE string
	collisionMsg string

	// Viewport
	width, height int

	// Timing
	lastUpdate time.Time
	ticker     *time.Ticker
}

type lane struct {
	y         int
	direction int // -1 for left, 1 for right
	speed     float64
}

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
		lives:          3,
		width:          80,
		height:         24,
		lastUpdate:     time.Now(),
		waveDuration:   15 * time.Second,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadVulnerabilities(),
		tea.EnterAltScreen,
		m.tick(), // Start ticking immediately for spinner animation
	)
}

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
	return tea.Tick(time.Second/60, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
