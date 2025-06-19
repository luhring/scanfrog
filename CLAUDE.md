# Scanfrog Development Guide (for Claude and humans!)

## Project Overview

Scanfrog is a terminal-based Frogger game that visualizes container vulnerabilities. It transforms vulnerability scan results from [Grype](https://github.com/anchore/grype) into an engaging game where players dodge CVE "obstacles" to showcase how secure (or vulnerable) container images are.

Additional context available in @./README.md.

## Key Architecture Decisions

### Vulnerability Data Flow
- **Grype Integration**: The game can either scan images in real-time (Mode A) or load pre-existing Grype JSON output (Mode B)
- **CVSS Score Handling**: Different vulnerability feeds provide different data:
  - Ubuntu/Debian feeds: Only severity labels (Critical/High/Medium/Low/Negligible), no CVSS scores
  - Alpine/Wolfi/Chainguard feeds: Include CVSS scores from NVD
  - The game gracefully handles both cases - never make up CVSS scores!

### Game Mechanics
- **Obstacle Mapping**: 
  - Critical/CVSS 9.0+ → T-Rex emoji 🦖 (wider, faster)
  - High/CVSS 7.0+ → Truck emoji 🚛 (wider, fast)
  - Medium/Low/Negligible → Car emoji 🚗 (normal size/speed)
- **Wave System**: Large vulnerability sets (>50) are split into waves
- **Difficulty Scaling**: Based on actual vulnerability severity, not artificially inflated

## Development Patterns

### Terminal UI Considerations
- **Adaptive Colors**: All styles use Lipgloss AdaptiveColor for both light and dark terminals
- **Double-Width Characters**: Emojis take 2 character spaces - the rendering loop accounts for this
- **Window Sizing**: Game board height accounts for 2-line header, bordered screens leave margin

### Testing Approach
- **Test Data**: `testdata/sample-vulns.json` contains example Grype output
- **Manual Testing**: Test with various images to see different vulnerability feeds:
  ```bash
  ./scanfrog ubuntu:latest     # No CVSS scores
  ./scanfrog alpine:latest     # Has CVSS scores
  ./scanfrog node:latest       # Mix of severities
  ```

## Common Pitfalls to Avoid

1. **Don't Estimate CVSS Scores**: If Grype doesn't provide a score, just use the severity label
2. **Emoji Rendering**: Always skip the next character position after rendering an emoji
3. **Finish Line Visibility**: The 'F' in "FINISH" can conflict with frog detection - we check coordinates
4. **Border Calculations**: Lipgloss Width/Height includes the border - account for this in sizing

## Debugging Tips

- **Grype Output Structure**: Check actual JSON with `grype -o json <image> | jq`
- **Severity Values**: Can be "Critical", "High", "Medium", "Low", "Negligible" (case-sensitive)
- **CVSS Location**: Sometimes at `cvss[].baseScore`, sometimes at `cvss[].metrics.baseScore`
- **Motion Smoothness**: Obstacles use floating-point positions for smooth movement

## Future Enhancements

Consider these areas for improvement:
- Power-ups based on security tools (e.g., temporary invincibility from "security scanning")
- Multiplayer mode for comparing different base images
- Historical vulnerability tracking (show improvement over time)
- Integration with other scanners beyond Grype
