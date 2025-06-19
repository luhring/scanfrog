# Obstacle Density Issue - Session Notes

## Problem Statement
When running scanfrog with a high-vulnerability image (e.g., wordpress with 471 vulnerabilities), the initial game load shows a sparse obstacle field (~50-80 obstacles visible), but after losing/winning and restarting (without exiting the app), the obstacle field becomes extremely dense (~200+ obstacles visible). The initial load should show the same dense field as after restart.

## Expected Behavior
- For 471 vulnerabilities, we expect to see a very dense obstacle field that makes the game nearly impossible
- Most or all vulnerabilities should be represented as obstacles ON THE SCREEN at all times
- The obstacle density should properly convey the security risk of the image
- Initial load and post-restart should show identical obstacle density

## Design Constraints (Agreed Upon)
1. **One-to-one mapping**: Each obstacle represents exactly one vulnerability - no inflation or deflation of numbers
2. **Even distribution**: Obstacles should be evenly spread across all 8 lanes
3. **No waves**: All vulnerabilities should be active as obstacles simultaneously
4. **Consistent behavior**: Initial load and restart should generate identical obstacle fields

## What I Tried

### Attempt 1: Window Size Synchronization
**Hypothesis**: Game was starting before receiving proper window size
**Implementation**: Added `windowSizeReceived` flag to delay game start until after WindowSizeMsg
**Result**: No improvement - sparse initial load persisted

### Attempt 2: Dynamic Density Scaling
**Hypothesis**: Need to adjust obstacle spacing based on vulnerability count
**Implementation**: Created formula to calculate spacing based on target visible ratio (40% of obstacles)
**Result**: No improvement - still sparse on initial load

### Attempt 3: Force Regeneration on Window Size Change
**Hypothesis**: Obstacles generated with default width=80 need regeneration when real width arrives
**Implementation**: Added `needsObstacleRegen` flag and regeneration logic in Update()
**Result**: No improvement

### Attempt 4: Track Generation Width
**Hypothesis**: Need to detect when obstacles were generated with wrong width
**Implementation**: Added `obstacleGenWidth` tracking and regeneration when width differs
**Result**: No improvement

### Attempt 5: Fixed Obstacle Count Based on Vulnerability Count
**Hypothesis**: Should generate based on vuln count, not screen width
**Implementation**: Set fixed obstacles per lane based on vulnerability count (100 per lane for 400+ vulns)
**Result**: Created too many obstacles (800 for 471 vulns) - violated one-to-one mapping constraint

### Attempt 6: Simplified Generation with Modulo Wrapping
**Hypothesis**: Create exactly 471 obstacles with tight spacing and loop wrapping
**Implementation**: One obstacle per vulnerability, distributed evenly, with modulo wrapping for continuous loop
**Result**: Still sparse (~50-60 visible instead of 150+)

## Key Observations

1. **Default width is 80**: Model initializes with width=80, height=24
2. **Actual terminal width**: Appears to be ~150+ based on screenshots
3. **After restart**: Shows extremely dense field, suggesting the generation logic CAN work
4. **The math**: With 471 vulns, 8 lanes, 8-pixel spacing:
   - ~59 obstacles per lane
   - ~472 pixels of obstacles per lane
   - On 150-pixel screen, should see ~19 per lane (152 total)

## Critical Unknown
Why does restart generate a dense field while initial load does not, given that both should be using the same width and generation logic?

## Facts (Not Guesses)
- Initial load shows ~50-80 obstacles
- Post-restart shows ~200+ obstacles  
- The generation function is called in both cases
- Terminal width should be known in both cases (after WindowSizeMsg)
- No error messages or build failures

## Next Steps for Investigation
1. Add logging to track exact values during generation (width, vuln count, obstacles generated)
2. Compare the exact sequence of events between initial load and restart
3. Check if obstacles are being generated but positioned off-screen
4. Verify that restart actually uses the same generation function