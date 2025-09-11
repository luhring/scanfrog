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
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
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
	// For SSH mode, we'll default to using sample data for now
	// This could be extended to allow SSH clients to specify images
	vulnSource := &grype.FileSource{Path: "testdata/sample-vulns.json"}

	// Create new game model for this session
	model := game.NewModel(vulnSource)

	return model, []tea.ProgramOption{tea.WithAltScreen()}
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start SSH server for remote scanfrog access",
	Long: `Start an SSH server that allows remote users to connect and play the scanfrog game.
Users can connect via SSH and play the vulnerability visualization game remotely.`,
	Example: `
  scanfrog serve                           # Start server on localhost:2222
  scanfrog serve --port 2223               # Use custom port
  scanfrog serve --host-key ./mykey.pem    # Use custom host key`,
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
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
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
