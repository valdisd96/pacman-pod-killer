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

## SARSA AI (`internal/ai/sarsa.go`, `state.go`, `qtable.go`)

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

This creates only **~400 possible states** (5x5x2^4), making learning fast.

### The Q-Table

The Q-Table is the "brain" - a lookup table that stores:

```
State -> [Q-value for Up, Q-value for Down, Q-value for Left, Q-value for Right]
```

Higher Q-value = better action. For example:
```
State: "Player is to the right, no walls"
Q-values: [Up: 0.5, Down: 0.3, Left: -1.0, Right: 8.5]  <- Go RIGHT!
```

### The Reward System

Enemies learn via rewards/punishments (defined in `internal/ai/sarsa.go`):

| Event | Reward |
|-------|--------|
| **Catch player** | +100 |
| Move **closer** to player | +1 |
| Move **farther** from player | -0.5 |
| **Hit a wall** | -2 |
| Every step (penalty) | -0.1 |

### The Learning Loop

Every tick, each enemy:

1. **Observe** current state (where is player? what moves are valid?)
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

### Persistence

The Q-Table is saved to `~/.pacman-pod-killer/qtable.json`, so enemies get smarter over multiple game sessions.

## Visual Flow

```
+----------------------------------------------------------+
|                    SARSA Learning Loop                   |
+----------------------------------------------------------+
|                                                          |
|  +---------+   +----------+   +---------+                |
|  |  State  |-->|  Choose  |-->|  Take   |                |
|  | (dx,dy, |   |  Action  |   | Action  |                |
|  |  walls) |   | e-greedy |   | (move)  |                |
|  +---------+   +----------+   +----+----+                |
|       ^                            |                     |
|       |        +---------+         |                     |
|       |        | Update  |<--------+                     |
|       |        | Q-Table |    Reward:                    |
|       |        |  (SARSA |    +100 catch                 |
|       +--------|formula) |    +1 closer                  |
|    Next State  +---------+    -0.5 farther               |
|                               -2 wall hit                |
+----------------------------------------------------------+
```

## File Structure

| File | Purpose |
|------|---------|
| `internal/ai/random.go` | Simple random movement AI |
| `internal/ai/state.go` | Defines what enemies "see" (~400 states) |
| `internal/ai/qtable.go` | Stores learned Q-values, save/load to JSON |
| `internal/ai/sarsa.go` | The learning algorithm and decision-making |
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
