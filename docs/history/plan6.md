# Implementation Plan: Enhanced Enemy AI (Option A)

**Date:** 2026-02-02
**Issue:** #7 - Add new label for RL
**Approach:** Extended Q-Table (Option A) - "Big Memory Book"

## Goals

Add two new state features to enemy AI:
1. **Bullet Detection:** Detect bullets within 7 cells approaching from any direction
2. **Open Space Counting:** Count empty cells (0-3) in each direction for escape route planning

## State Space Analysis

- **Before:** ~400 states (5×5 player position × 16 wall sensors)
- **After:** ~1.6M states (5×5 × 16 × 16 bullet directions × 16 open space counts)
- **Impact:** 4000x increase in state space, requiring more training time but enabling smarter behaviors

## Implementation Phases

### Phase 1: Utility Modules

**Files:** `internal/ai/bullet.go`, `internal/ai/openspace.go`

Create standalone detection utilities:
- `DetectBulletThreat()`: Check for bullets within 7 cells moving toward enemy
- `CountOpenSpace()`: Ray-cast up to 3 cells in each direction counting walkable space

### Phase 2: State Representation Update

**File:** `internal/ai/state.go`

Add to `State` struct:
```go
BulletDir [4]bool  // Threat from up/down/left/right
OpenDir   [4]int   // Open cells 0-3 in each direction
```

Update `ToKey()` to pack 256x more state combinations.

### Phase 3: Reward System Enhancement

**File:** `internal/ai/sarsa.go`

New reward constants:
```go
RewardBulletThreat = -8.0   // Strong bullet avoidance
RewardOpenSpace    = +0.5   // Prefer open areas  
RewardDeadEnd      = -2.0   // Avoid getting trapped
```

Update `computeReward()` to evaluate bullet danger and space availability.

### Phase 4: Controller Integration

**File:** `internal/game/controller.go`

Update `moveEnemies()` to pass bullet positions to AI.

### Phase 5: Q-Table Versioning

**File:** `internal/ai/qtable.go`

Add version field to Q-table JSON. Start fresh on version mismatch.

## Configuration Parameters

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Bullet detection range | 7 cells | Gives time to react but not too early |
| Open space depth | 3 cells | Enough to see dead-ends without overcomplicating |
| Bullet threat reward | -8.0 | Strong enough to prioritize dodging |
| Open space reward | +0.5 | Mild preference for escape routes |
| Dead end penalty | -2.0 | Moderate avoidance of traps |

## Expected Behaviors

1. **Bullet Dodging:** Enemies detect bullets approaching and move perpendicular or retreat
2. **Space Awareness:** Enemies prefer corridors with multiple exits over dead-ends
3. **Smarter Pursuit:** Enemies won't chase player into narrow corridors if blocked by bullets
4. **Flanking:** With open space awareness, enemies may try to surround player from multiple directions

## Trade-offs

**Pros:**
- Smarter, more human-like enemy behavior
- Enemies survive longer against shooting player
- More engaging gameplay

**Cons:**
- Longer training time needed (more states to explore)
- Higher memory usage for Q-table
- May need 5-10 games before enemies show smart behavior

## Testing Strategy

1. Unit tests for bullet detection logic
2. Unit tests for open space counting
3. Integration tests for new state key generation
4. Manual playtesting to verify dodging behavior

## Versioning

Q-table format version: 2
- Version 1: Original state (400 states)
- Version 2: Enhanced state with bullets + open space

On version mismatch, discard old Q-table and start fresh training.
