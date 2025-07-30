// Package main provides the command-line interface for the scanfrog vulnerability visualization game.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luhring/scanfrog/internal/game"
	"github.com/luhring/scanfrog/internal/grype"
	"github.com/spf13/cobra"
)

const devVersion = "dev"

var (
	// Version information set by goreleaser during build
	version = devVersion
	commit  = "none"
	date    = "unknown"

	jsonFile string
	rootCmd  = &cobra.Command{
		Use:   "scanfrog [image]",
		Short: "A Frogger-style game visualizing container vulnerabilities",
		Long: `scanfrog is a terminal game that visualizes container vulnerabilities
as obstacles in a Frogger-style play-field. Vulnerabilities are discovered
using Grype and rendered with Bubble Tea.`,
		Example: `  scanfrog ubuntu:latest         # Scan an image with Grype
  scanfrog --json results.json   # Load from Grype JSON file`,
		Args: cobra.MaximumNArgs(1),
		RunE: runGame,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("scanfrog version %s\n", version)
			if version != devVersion {
				fmt.Printf("  commit: %s\n", commit)
				fmt.Printf("  built:  %s\n", date)
			}
		},
	}
)

func init() {
	rootCmd.Flags().StringVar(&jsonFile, "json", "", "Path to pre-existing Grype JSON file")
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runGame(_ *cobra.Command, args []string) error {
	if jsonFile == "" && len(args) == 0 {
		return fmt.Errorf("must specify either an image to scan or --json file")
	}

	if jsonFile != "" && len(args) > 0 {
		return fmt.Errorf("cannot specify both image and --json file")
	}

	var vulnSource grype.VulnerabilitySource
	if jsonFile != "" {
		vulnSource = &grype.FileSource{Path: jsonFile}
	} else {
		vulnSource = &grype.ScannerSource{Image: args[0]}
	}

	model := game.NewModel(vulnSource)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run game: %w", err)
	}

	return nil
}
