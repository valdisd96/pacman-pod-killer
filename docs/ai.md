# AI System Documentation

This document explains how the AI system works in Pacman Pod Killer.

## Overview

The project has **two AI modes** for controlling enemies (ghosts):

1. **Random AI** - Simple random movement
2. **SARSA AI** - Reinforcement learning that improves over time

## Random AI (`internal/ai/random.go`)

The simplest approach:
- Pick a random direction (up, down, left, right)
- Check if it's a valid move (no walls)
- Move that way

```
Enemy -> Shuffle directions -> Pick first valid one -> Move
```

It's simple but not smart - enemies just wander aimlessly.

## SARSA AI (`internal/ai/sarsa.go`, `state.go`, `qtable.go`, `bullet.go`, `openspace.go`)

This is **reinforcement learning** - the enemy learns from experience how to chase the player.

### Key Concept: Q-Learning Family

SARSA stands for **S**tate-**A**ction-**R**eward-**S**tate-**A**ction. It's a technique where the agent:
1. Observes current **State**
2. Takes an **Action**
3. Gets a **Reward**
4. Sees the new **State**
5. Picks the next **Action**

Then updates its knowledge based on this experience.

### State Representation

Each enemy "sees" a simplified version of the world (defined in `internal/ai/state.go`):

| Component | Description |
|-----------|-------------|
| `DeltaX` | Horizontal distance to player (clamped -2 to +2) |
| `DeltaY` | Vertical distance to player (clamped -2 to +2) |
| `CanUp/Down/Left/Right` | Which directions have no walls |
| `BulletUp/Down/Left/Right` | Bullet approaching from this direction (within 7 cells) |
| `OpenUp/Down/Left/Right` | Number of walkable cells (0-3) in each direction |

This creates approximately **1.6 million possible states** (5x5x16x16x256), enabling sophisticated behaviors like bullet dodging and dead-end avoidance.

#### Bullet Detection (`internal/ai/bullet.go`)

Enemies detect threats from bullets:
- **Range**: 7 cells in each cardinal direction (up/down/left/right)
- **Condition**: Only bullets moving toward the enemy are flagged
- **Diagonal bullets**: Ignored to keep state space manageable
- **Purpose**: Early warning system for dodging

The 7-cell range balances early detection with computational efficiency - far enough to react, but not so far that distant bullets cause unnecessary avoidance.

#### Open Space Evaluation (`internal/ai/openspace.go`)

Enemies evaluate escape routes using ray-casting:
- **Depth**: 3 cells in each direction
- **Count**: Walkable floor cells (0-3) from current position
- **Purpose**: Detect dead-ends and prefer corridors with escape routes

The 3-cell depth provides enough lookahead to identify trapped positions while keeping computation lightweight.

### The Q-Table

The Q-Table is the "brain" - a lookup table that stores:

```
State -> [Q-value for Up, Q-value for Down, Q-value for Left, Q-value for Right]
```

Higher Q-value = better action. For example:
```
State: "Player is to the right, no walls, bullet from left, 3 open cells ahead"
Q-values: [Up: 0.5, Down: 0.3, Left: -10.0, Right: 8.5]  <- Avoid left (bullet), go RIGHT!
```

### Q-Table Versioning (`internal/ai/qtable.go`)

The Q-table format is versioned to handle state space changes:

- **Version 2.0** (current): Enhanced state with bullet threats and open space (~1.6M states)
- **Version 1.0** (legacy): Basic state with position and walls only (~400 states)

On load, if the version doesn't match `QTableVersion` (2.0), the Q-table is reset to empty. This ensures old incompatible learning data doesn't corrupt the new AI behavior.

### The Reward System

Enemies learn via rewards/punishments (defined in `internal/ai/sarsa.go`):

| Event | Reward | Rationale |
|-------|--------|-----------|
| **Catch player** | +100 | Ultimate goal |
| Move **closer** to player | +1 | Encourage pursuit |
| Move **farther** from player | -0.5 | Discourage fleeing |
| **Hit a wall** | -2 | Punish invalid moves |
| Every step (penalty) | -0.1 | Encourage efficiency |
| **Bullet threat** | -8 | Strong avoidance of bullets moving toward enemy |
| **Open space** | +0.5 per cell | Prefer corridors with escape routes |
| **Dead end** | -2 | Penalty for entering trapped positions (0 open cells) |

The bullet threat reward (-8) is particularly strong to ensure survival - it's better to take a longer path than move toward a bullet. The open space reward encourages enemies to stay in areas with multiple escape routes.

### The Learning Loop

Every tick, each enemy:

1. **Observe** current state:
   - Where is the player?
   - What moves are valid?
   - Are there approaching bullets?
   - How much open space is in each direction?
2. **Choose action** via epsilon-greedy:
   - 10% of time: pick **random** (explore new strategies)
   - 90% of time: pick **best known** action (exploit learning)
3. **Update Q-Table** using the SARSA formula:
   ```
   Q(s,a) += alpha * [reward + gamma * Q(s',a') - Q(s,a)]
   ```
   Where:
   - `alpha` = 0.1 - learning rate (how fast to update)
   - `gamma` = 0.95 - discount factor (how much future rewards matter)

### Epsilon-Greedy Exploration

The `epsilon` parameter (default 0.1) controls exploration vs exploitation:

- With probability `epsilon`: pick a **random** valid action (explore)
- With probability `1 - epsilon`: pick the **best** action according to Q-table (exploit)

This balance ensures enemies:
- Try new strategies sometimes (might find better paths)
- Usually use what they've already learned works

### Collective Learning

All enemies share a **single Q-table**. This means:
- Learning from one enemy benefits all others
- The collective improves faster than individual enemies
- The Q-table represents "general ghost knowledge" about chasing the player

### Persistence

The Q-Table is saved to `~/.pacman-pod-killer/qtable.json`, so enemies get smarter over multiple game sessions. The file includes:
- Version number (2.0)
- All learned state-action values
- Automatic reset on version mismatch

## Visual Flow

```
+----------------------------------------------------------+
|                    SARSA Learning Loop                   |
+----------------------------------------------------------+
|                                                          |
|  +---------+   +----------+   +---------+                |
|  |  State  |-->|  Choose  |-->|  Take   |                |
|  |(dx,dy, |   |  Action  |   | Action  |                |
|  | walls,  |   | e-greedy |   | (move)  |                |
|  |bullets, |   |          |   |         |                |
|  |  open)  |   |          |   |         |                |
|  +---------+   +----------+   +----+----+                |
|       ^                            |                     |
|       |        +---------+         |                     |
|       |        | Update  |<--------+                     |
|       |        | Q-Table |    Reward:                    |
|       |        |  (SARSA |    +100 catch                 |
|       +--------|formula) |    +1 closer                  |
|    Next State  +---------+    -0.5 farther               |
|                               -2 wall hit                |
|                               -8 bullet threat           |
|                               +0.5 open space            |
|                               -2 dead end                |
+----------------------------------------------------------+
```

## File Structure

| File | Purpose |
|------|---------|
| `internal/ai/random.go` | Simple random movement AI |
| `internal/ai/state.go` | Defines what enemies "see" (~1.6M states) |
| `internal/ai/qtable.go` | Stores learned Q-values, save/load to JSON, version tracking |
| `internal/ai/sarsa.go` | The learning algorithm and decision-making |
| `internal/ai/bullet.go` | Bullet detection for threat avoidance |
| `internal/ai/openspace.go` | Open space evaluation for dead-end avoidance |
| `internal/ai/sarsa_test.go` | Unit tests for the AI system |

## Configuration Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--ai-mode` | `sarsa` | AI mode: `random` or `sarsa` |
| `--training` | `true` | Enable SARSA training (updates Q-table) |
| `--epsilon` | `0.1` | Exploration rate (0.0-1.0) |
| `--qtable` | `~/.pacman-pod-killer/qtable.json` | Path to Q-table file |

## Tips

- The more you play with `--training=true`, the smarter enemies become
- Set `--epsilon=0` to disable exploration (pure exploitation)
- Delete the Q-table file to reset enemy learning
- Use `--ai-mode=random` if you want simple, non-learning enemies
- Enemies will learn to dodge bullets and avoid dead-ends over time
- The 7-cell bullet detection range gives enemies time to react without overreacting to distant threats
