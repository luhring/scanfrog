# Scanfrog 🐸

A Frogger-style terminal game that visualizes container vulnerabilities discovered by Grype.

## Features

- Scan container images for vulnerabilities using Grype
- Visualize vulnerabilities as obstacles in a Frogger-style game
- Difficulty scales with vulnerability count and severity
- Beautiful terminal UI powered by Bubble Tea and Lip Gloss

## Installation

```bash
go install github.com/luhring/scanfrog/cmd/scanfrog@latest
```

## Usage

### Mode A: Scan an image
```bash
scanfrog ubuntu:latest
```

### Mode B: Load from Grype JSON
```bash
# First generate a Grype report
grype ubuntu:latest -o json > vulns.json

# Then play the game
scanfrog --json vulns.json
```

## Controls

- **↑/W** - Move up
- **↓/S** - Move down  
- **←/A** - Move left
- **→/D** - Move right
- **Q/Esc** - Quit

## Game Mechanics

### Obstacle Types
- **Normal car** (CVSS < 4) - Regular speed, single width
- **Fast car** (CVSS 4-7) - Increased speed
- **Truck** (CVSS 7-9) - Double width, faster
- **Boss** (CVSS ≥ 9) - Double width, very fast, blinking

### Difficulty Scaling
- 0 vulns → Instant win
- 5-30 vulns → Casual difficulty
- ~100 vulns → Challenging
- 1000+ vulns → Up to 10 minutes of gameplay

## Development

### Requirements
- Go 1.24+
- Grype installed (for Mode A)

### Building
```bash
go build ./cmd/scanfrog
```

### Testing
```bash
# Use the sample vulnerability data
./scanfrog --json testdata/sample-vulns.json
```

## License

Apache License 2.0