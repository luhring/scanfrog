package grype

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSource_GetVulnerabilities(t *testing.T) {
	tests := []struct {
		name       string
		jsonFile   string
		wantErr    bool
		minVulns   int
		checkVulns func(t *testing.T, vulns []Vulnerability)
	}{
		{
			name:     "valid sample vulnerabilities",
			jsonFile: "../../testdata/sample-vulns.json",
			wantErr:  false,
			minVulns: 1,
			checkVulns: func(t *testing.T, vulns []Vulnerability) {
				// Check that we parsed at least some vulnerabilities
				if len(vulns) == 0 {
					t.Error("expected at least one vulnerability")
				}

				// Check first vulnerability has required fields
				if len(vulns) > 0 {
					v := vulns[0]
					if v.ID == "" {
						t.Error("vulnerability ID should not be empty")
					}
					if v.Severity == "" {
						t.Error("vulnerability severity should not be empty")
					}
				}

				// Check severity values are valid
				validSeverities := map[string]bool{
					"Critical":   true,
					"High":       true,
					"Medium":     true,
					"Low":        true,
					"Negligible": true,
				}
				for _, v := range vulns {
					if !validSeverities[v.Severity] {
						t.Errorf("invalid severity value: %s", v.Severity)
					}
				}
			},
		},
		{
			name:     "non-existent file",
			jsonFile: "does-not-exist.json",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &FileSource{Path: tt.jsonFile}
			vulns, err := fs.GetVulnerabilities()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetVulnerabilities() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(vulns) < tt.minVulns {
					t.Errorf("expected at least %d vulnerabilities, got %d", tt.minVulns, len(vulns))
				}
				if tt.checkVulns != nil {
					tt.checkVulns(t, vulns)
				}
			}
		})
	}
}

func TestParseGrypeOutput(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
		wantErr     bool
		wantVulns   int
	}{
		{
			name: "valid JSON with vulnerabilities",
			jsonContent: `{
				"matches": [
					{
						"vulnerability": {
							"id": "CVE-2021-12345",
							"severity": "High",
							"description": "Test vulnerability",
							"cvss": []
						},
						"artifact": {
							"name": "test-package",
							"version": "1.0.0"
						}
					}
				]
			}`,
			wantErr:   false,
			wantVulns: 1,
		},
		{
			name: "valid JSON with CVSS score",
			jsonContent: `{
				"matches": [
					{
						"vulnerability": {
							"id": "CVE-2021-12345",
							"severity": "Critical",
							"description": "Critical vulnerability",
							"cvss": [
								{
									"metrics": {
										"baseScore": 9.8
									}
								}
							]
						},
						"artifact": {
							"name": "critical-package",
							"version": "2.0.0"
						}
					}
				]
			}`,
			wantErr:   false,
			wantVulns: 1,
		},
		{
			name:        "invalid JSON",
			jsonContent: `{invalid json`,
			wantErr:     true,
		},
		{
			name:        "empty matches",
			jsonContent: `{"matches": []}`,
			wantErr:     false,
			wantVulns:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vulns, err := parseGrypeOutput([]byte(tt.jsonContent))

			if (err != nil) != tt.wantErr {
				t.Errorf("parseGrypeOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(vulns) != tt.wantVulns {
					t.Errorf("expected %d vulnerabilities, got %d", tt.wantVulns, len(vulns))
				}
			}
		})
	}
}

func TestCVSSScoreParsing(t *testing.T) {
	jsonContent := `{
		"matches": [
			{
				"vulnerability": {
					"id": "CVE-2021-1",
					"severity": "High",
					"cvss": [{"baseScore": 7.5}]
				},
				"artifact": {"name": "pkg1", "version": "1.0"}
			},
			{
				"vulnerability": {
					"id": "CVE-2021-2",
					"severity": "Critical",
					"cvss": [{"metrics": {"baseScore": 9.0}}]
				},
				"artifact": {"name": "pkg2", "version": "1.0"}
			},
			{
				"vulnerability": {
					"id": "CVE-2021-3",
					"severity": "Medium",
					"cvss": []
				},
				"artifact": {"name": "pkg3", "version": "1.0"}
			}
		]
	}`

	vulns, err := parseGrypeOutput([]byte(jsonContent))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedScores := []float64{7.5, 9.0, 0}
	if len(vulns) != len(expectedScores) {
		t.Fatalf("expected %d vulnerabilities, got %d", len(expectedScores), len(vulns))
	}

	for i, expected := range expectedScores {
		if vulns[i].CVSS != expected {
			t.Errorf("vuln %d: expected CVSS score %f, got %f", i, expected, vulns[i].CVSS)
		}
	}
}

// TestMain ensures test data exists
func TestMain(m *testing.M) {
	// Verify test data exists
	testFile := filepath.Join("..", "..", "testdata", "sample-vulns.json")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		panic("test data file not found: " + testFile)
	}

	os.Exit(m.Run())
}
