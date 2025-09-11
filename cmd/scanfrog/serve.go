package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
	"github.com/luhring/scanfrog/internal/game"
	"github.com/luhring/scanfrog/internal/grype"
	"github.com/spf13/cobra"
)

var (
	sshPort     int
	hostKeyPath string
)

// teaHandler creates a new Bubble Tea program for each SSH session
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	// Get the command from the SSH session - this is what the user typed after the hostname
	// e.g., ssh -p 2222 localhost ubuntu:latest -> command = ["ubuntu:latest"]
	command := s.Command()

	var vulnSource grype.VulnerabilitySource

	if len(command) == 0 {
		// No command provided - use sample data
		vulnSource = &grype.FileSource{Path: "testdata/sample-vulns.json"}
	} else {
		// Use the first argument as the image name to scan
		imageName := command[0]
		vulnSource = &grype.ScannerSource{Image: imageName}
	}

	// Create new game model for this session
	model := game.NewModel(vulnSource)

	return model, []tea.ProgramOption{tea.WithAltScreen()}
}

// customSessionHandler handles SSH sessions with both PTY and non-PTY support
func customSessionHandler(next ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		command := s.Command()

		// Check if we have a PTY
		_, winCh, isPty := s.Pty()

		if isPty {
			// Handle PTY session (interactive) - use Bubble Tea
			model, opts := teaHandler(s)

			// Add PTY-specific options
			opts = append(opts, tea.WithInput(s), tea.WithOutput(s))

			p := tea.NewProgram(model, opts...)

			// Handle window resize events
			go func() {
				for win := range winCh {
					p.Send(tea.WindowSizeMsg{
						Width:  win.Width,
						Height: win.Height,
					})
				}
			}()

			if _, err := p.Run(); err != nil {
				fmt.Fprintf(s.Stderr(), "Error running game: %v\r\n", err)
			}
		} else {
			// Handle non-PTY session (command execution)
			if len(command) == 0 {
				// No command and no PTY - instruct user to use PTY for interactive mode
				fmt.Fprintf(s, "Interactive mode requires a PTY. Use: ssh -t -p %d localhost\r\n", sshPort)
				fmt.Fprintf(s, "Or specify an image to scan: ssh -p %d localhost ubuntu:latest\r\n", sshPort)
				return
			}

			// For command execution, we'll provide a text-based response
			imageName := command[0]
			fmt.Fprintf(s, "Scanning image: %s\r\n", imageName)
			fmt.Fprintf(s, "Note: This would normally run the interactive game.\r\n")
			fmt.Fprintf(s, "For the full interactive experience, use: ssh -t -p %d localhost %s\r\n", sshPort, imageName)
		}
	}
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start SSH server for remote scanfrog access",
	Long: `Start an SSH server that allows remote users to connect and play the scanfrog game.
Users can connect via SSH and specify an image to scan, or use sample data if no image is provided.`,
	Example: `  # Start server
  scanfrog serve                           # Start server on localhost:2222
  scanfrog serve --port 2223               # Use custom port
  scanfrog serve --host-key ./mykey.pem    # Use custom host key

  # Connect and play (from another terminal)
  ssh -t -p 2222 localhost                # Play with sample vulnerabilities
  ssh -t -p 2222 localhost ubuntu:latest  # Scan and play with ubuntu:latest
  ssh -p 2222 localhost alpine:3.18       # Get scan info (non-interactive)`,
	SilenceUsage: true,
	RunE:         runServe,
}

func init() {
	// Set default host key path
	homeDir, _ := os.UserHomeDir()
	defaultKeyPath := filepath.Join(homeDir, ".ssh", "scanfrog_host_key")

	serveCmd.Flags().IntVar(&sshPort, "port", 2222, "Port to bind SSH server to")
	serveCmd.Flags().StringVar(&hostKeyPath, "host-key", defaultKeyPath, "Path to SSH host key (will be generated if it doesn't exist)")
}

func runServe(*cobra.Command, []string) error {
	server, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf(":%d", sshPort)),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithMiddleware(
			customSessionHandler,
			logging.Middleware(),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create SSH server: %w", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Starting SSH server on :%d\n", sshPort)
	fmt.Printf("Users can connect with: ssh -p %d localhost\n", sshPort)

	go func() {
		if err = server.ListenAndServe(); err != nil && err != ssh.ErrServerClosed {
			fmt.Printf("SSH server error: %v\n", err)
		}
	}()

	<-done
	fmt.Println("\nShutting down SSH server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil && err != ssh.ErrServerClosed {
		return fmt.Errorf("SSH server shutdown error: %w", err)
	}

	fmt.Println("SSH server stopped")
	return nil
}
