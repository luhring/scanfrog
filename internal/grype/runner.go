// Package grype provides interfaces and implementations for vulnerability scanning using the Grype tool.
package grype

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// VulnerabilitySource is an interface for getting vulnerabilities
type VulnerabilitySource interface {
	GetVulnerabilities() ([]Vulnerability, error)
}

// Vulnerability represents a single CVE from Grype output
type Vulnerability struct {
	ID          string  `json:"id"`
	Severity    string  `json:"severity"`
	CVSS        float64 `json:"cvss"`
	Package     string  `json:"package"`
	Description string  `json:"description"`
}

// Output represents the JSON structure from Grype
type Output struct {
	Matches []Match `json:"matches"`
}

// Match represents a vulnerability match in Grype output
type Match struct {
	Vulnerability VulnerabilityInfo `json:"vulnerability"`
	Artifact      ArtifactInfo      `json:"artifact"`
}

// VulnerabilityInfo contains vulnerability details
type VulnerabilityInfo struct {
	ID          string     `json:"id"`
	Severity    string     `json:"severity"`
	Description string     `json:"description"`
	CVSS        []CVSSInfo `json:"cvss"`
}

// CVSSInfo contains CVSS score information
type CVSSInfo struct {
	Source  string      `json:"source"`
	Type    string      `json:"type"`
	Score   float64     `json:"baseScore"`
	Metrics CVSSMetrics `json:"metrics"`
}

// CVSSMetrics contains nested CVSS metrics
type CVSSMetrics struct {
	BaseScore float64 `json:"baseScore"`
}

// ArtifactInfo contains package information
type ArtifactInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ScannerSource runs Grype on an image
type ScannerSource struct {
	Image string
}

// GetVulnerabilities runs Grype and returns vulnerabilities
func (s *ScannerSource) GetVulnerabilities() ([]Vulnerability, error) {
	// Validate the image name to prevent command injection
	if err := validateImageName(s.Image); err != nil {
		return nil, fmt.Errorf("invalid image name: %w", err)
	}

	// #nosec G204 -- Image name has been validated above to prevent command injection
	cmd := exec.Command("grype", s.Image, "-o", "json", "-q")
	output, err := cmd.Output()
	if err != nil {
		// If it's an exec error, try to get stderr for better error message
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("grype failed: %s", exitErr.Stderr)
		}
		return nil, fmt.Errorf("failed to run grype: %w", err)
	}

	return parseGrypeOutput(output)
}

// FileSource reads vulnerabilities from a JSON file
type FileSource struct {
	Path string
}

// GetVulnerabilities reads and parses a Grype JSON file
func (f *FileSource) GetVulnerabilities() ([]Vulnerability, error) {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return parseGrypeOutput(data)
}

func parseGrypeOutput(data []byte) ([]Vulnerability, error) {
	var output Output
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	vulns := make([]Vulnerability, 0, len(output.Matches))
	for _, match := range output.Matches {
		vuln := Vulnerability{
			ID:          match.Vulnerability.ID,
			Severity:    match.Vulnerability.Severity,
			Package:     match.Artifact.Name,
			Description: match.Vulnerability.Description,
		}

		// Get highest CVSS score if available
		for _, cvss := range match.Vulnerability.CVSS {
			// Try to get score from either top level or metrics
			score := cvss.Score
			if score == 0 && cvss.Metrics.BaseScore > 0 {
				score = cvss.Metrics.BaseScore
			}
			if score > vuln.CVSS {
				vuln.CVSS = score
			}
		}

		vulns = append(vulns, vuln)
	}

	return vulns, nil
}

// validateImageName checks if the image name is safe to use in a command
func validateImageName(image string) error {
	if image == "" {
		return fmt.Errorf("image name cannot be empty")
	}

	// Check for shell metacharacters that could be used for command injection
	dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]", "<", ">", "\n", "\r", "\\"}
	for _, char := range dangerousChars {
		if strings.Contains(image, char) {
			return fmt.Errorf("image name contains invalid character: %s", char)
		}
	}

	// Additional validation: image names should follow Docker naming conventions
	// They can contain lowercase letters, digits, underscores, periods, and dashes
	// They can also have a registry prefix and tag suffix
	// Examples: ubuntu:latest, docker.io/library/nginx:1.21, my-app:v1.0.0
	// This regex allows for more complex image names while still being safe
	validImageRegex := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-/:]*(:[a-zA-Z0-9._\-]+)?(@sha256:[a-f0-9]+)?$`)
	if !validImageRegex.MatchString(image) {
		return fmt.Errorf("image name contains invalid format")
	}

	return nil
}
