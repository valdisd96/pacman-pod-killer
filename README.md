# Pacman Pod Killer

Terminal Pacman game that maps running Docker containers to enemies. When you kill an enemy, its container is removed.

## Features

- Terminal-based Pacman with procedurally generated mazes
- Each running Docker container spawns an enemy ghost
- Collision with enemies kills both and removes the container
- **Shooting**: Fire bullets to kill enemies from a distance (also removes containers)
- **SARSA reinforcement learning**: Enemies learn to chase you over time
  - Bullet dodging: Enemies detect and avoid approaching bullets
  - Dead-end avoidance: Enemies prefer escape routes and avoid traps
  - Collective learning: Shared Q-table across all enemies

## Run

```bash
go run ./cmd/pacman
```

macOS Docker Desktop socket example:

```bash
DOCKER_HOST=unix:///Users/<username>/.docker/run/docker.sock \
script -q /dev/null go run ./cmd/pacman --width 80 --height 30
```

## Controls

- Arrow keys or WASD to move
- Spacebar to shoot (fires in direction of last movement)
- Q to quit

## AI Modes

The game supports two enemy AI modes:

### SARSA (Default)

Enemies use reinforcement learning to chase the player. They learn from experience and get smarter over time, with enhanced capabilities:

- **Bullet dodging**: Detects bullets within 7 cells and avoids moving toward them
- **Dead-end avoidance**: Evaluates open space (0-3 cells) in each direction to avoid getting trapped
- **Collective intelligence**: All enemies share a single Q-table for faster learning

```bash
# Default: SARSA with training enabled
go run ./cmd/pacman

# SARSA with higher exploration (faster learning)
go run ./cmd/pacman --epsilon 0.3

# Use trained Q-table without further learning
go run ./cmd/pacman --training=false --epsilon 0
```

### Random

Enemies move randomly with no learning.

```bash
go run ./cmd/pacman --ai-mode random
```

## Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--width` | 15 | Logical maze width (corridor cells) |
| `--height` | 10 | Logical maze height (corridor cells) |
| `--seed` | time-based | Random seed for maze generation |
| `--tick-ms` | 80 | Game tick duration in milliseconds |
| `--respawn-delay` | 10000 | Player respawn delay in milliseconds |
| `--debug` | false | Enable debug logging |
| `--log-file` | "" | Path to log file (empty = disabled) |
| `--ai-mode` | sarsa | Enemy AI: `random` or `sarsa` |
| `--training` | true | Enable SARSA learning (updates Q-table) |
| `--epsilon` | 0.1 | SARSA exploration rate (0.0-1.0) |
| `--qtable` | ~/.pacman-pod-killer/qtable.json | Path to Q-table file |

## SARSA Reinforcement Learning

The SARSA (State-Action-Reward-State-Action) implementation uses on-policy learning with enhanced state representation:

### State Space (~1.6M states)

- **Relative position**: Player position relative to enemy (clamped to [-2,+2])
- **Wall sensors**: Available movement directions (4 boolean values)
- **Bullet threats**: Detects bullets approaching from up/down/left/right within 7 cells
- **Open space**: Counts walkable cells (0-3) in each direction to detect dead-ends

### Actions

Up, Down, Left, Right (4 actions)

### Reward Function

| Reward | Value | Description |
|--------|-------|-------------|
| Catch player | +100 | Ultimate goal - enemy catches player |
| Move closer | +1 | Encourages pursuit behavior |
| Move farther | -0.5 | Discourages fleeing from player |
| Hit wall | -2 | Punishes invalid move attempts |
| Step penalty | -0.1 | Encourages efficient paths |
| Bullet threat | -8 | Strong avoidance of bullets moving toward enemy |
| Open space | +0.5 per cell | Prefers corridors with escape routes |
| Dead end | -2 | Penalty for entering trapped positions |

### Learning Features

- **Collective learning**: All enemies share a single Q-table
- **Persistent learning**: Q-table saved to `~/.pacman-pod-killer/qtable.json` between sessions
- **Version tracking**: Q-table format v2.0 (old v1.0 tables auto-discarded)
- **Bullet dodging**: Enemies learn to avoid moving toward detected bullet threats
- **Dead-end avoidance**: Enemies prefer paths with open escape routes

## Warning

This game deletes running Docker containers when enemies die. Use with caution.

## Build

```bash
go build ./cmd/pacman
```

## Test

```bash
go test ./...
```
