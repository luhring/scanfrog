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
	pos      position
	width    int
	speed    float64
	cveID    string
	severity float64
}

type Model struct {
	vulnSource grype.VulnerabilitySource
	state      gameState
	
	// Loading state
	loadingMsg string
	
	// Game state
	frog       position
	obstacles  []obstacle
	lanes      []lane
	score      int
	lives      int
	
	// Wave management
	currentWave  int
	totalWaves   int
	waveTimer    time.Time
	waveDuration time.Duration
	
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
	return &Model{
		vulnSource:   vulnSource,
		state:        stateLoading,
		loadingMsg:   "Loading vulnerabilities...",
		lives:        3,
		width:        80,
		height:       24,
		lastUpdate:   time.Now(),
		waveDuration: 15 * time.Second,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadVulnerabilities(),
		tea.EnterAltScreen,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	
	case vulnerabilitiesLoadedMsg:
		return m.startGame(msg.vulns), m.tick()
	
	case vulnerabilityErrorMsg:
		m.state = stateGameOver
		m.collisionMsg = fmt.Sprintf("Failed to load vulnerabilities: %v", msg.err)
		return m, tea.Quit
	
	case tickMsg:
		if m.state == statePlaying {
			return m.updateGame(), m.tick()
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
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}