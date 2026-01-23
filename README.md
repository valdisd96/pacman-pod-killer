# Pacman Pod Killer

Terminal Pacman game that maps running Docker containers to enemies. When you kill an enemy, its container is removed.

## Features

- Terminal-based Pacman with procedurally generated mazes
- Each running Docker container spawns an enemy ghost
- Collision with enemies kills both and removes the container
- **SARSA reinforcement learning**: Enemies learn to chase you over time

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
- Q to quit

## AI Modes

The game supports two enemy AI modes:

### SARSA (Default)

Enemies use reinforcement learning to chase the player. They learn from experience and get smarter over time.

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

The SARSA (State-Action-Reward-State-Action) implementation uses on-policy learning:

- **State**: Relative position to player + available movement directions (~400 states)
- **Actions**: Up, Down, Left, Right
- **Rewards**: +100 catch player, +1 move closer, -0.5 move farther, -2 hit wall
- **Learning**: Q-table saved between sessions for persistent learning

All enemies share a single Q-table, so learning from one enemy benefits all others.

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
