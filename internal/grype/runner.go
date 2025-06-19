package grype

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

// GrypeOutput represents the JSON structure from Grype
type GrypeOutput struct {
	Matches []Match `json:"matches"`
}

// Match represents a vulnerability match in Grype output
type Match struct {
	Vulnerability VulnerabilityInfo `json:"vulnerability"`
	Artifact      ArtifactInfo      `json:"artifact"`
}

// VulnerabilityInfo contains vulnerability details
type VulnerabilityInfo struct {
	ID          string   `json:"id"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	CVSS        []CVSSInfo `json:"cvss"`
}

// CVSSInfo contains CVSS score information
type CVSSInfo struct {
	Source string `json:"source"`
	Type   string `json:"type"`
	Score  float64 `json:"baseScore"`
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
	cmd := exec.Command("grype", s.Image, "-o", "json", "-q")
	output, err := cmd.Output()
	if err != nil {
		// If it's an exec error, try to get stderr for better error message
		if exitErr, ok := err.(*exec.ExitError); ok {
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
	var output GrypeOutput
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
		
		// Get highest CVSS score, preferring non-zero scores
		for _, cvss := range match.Vulnerability.CVSS {
			if cvss.Score > vuln.CVSS {
				vuln.CVSS = cvss.Score
			}
		}
		
		// If we still have no CVSS score, estimate based on severity
		if vuln.CVSS == 0 {
			switch vuln.Severity {
			case "Critical":
				vuln.CVSS = 9.0 // Estimate for critical
			case "High":
				vuln.CVSS = 7.5 // Estimate for high
			case "Medium":
				vuln.CVSS = 5.0 // Estimate for medium
			case "Low":
				vuln.CVSS = 2.5 // Estimate for low
			case "Negligible":
				vuln.CVSS = 0.5 // Estimate for negligible
			}
		}
		
		vulns = append(vulns, vuln)
	}
	
	return vulns, nil
}