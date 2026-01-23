# Implementation Plan: Terminal Pacman Docker Game

## Defaults & CLI
- Defaults: width=80, height=30, respawnDelay=10s, tick=80ms
- Flags: --width, --height, --seed, --respawn-delay, --tick-ms

## Project Layout (Go)
- cmd/pacman/main.go - CLI wiring, init, run loop
- internal/game - state, loop, collisions, respawn
- internal/maze - randomized maze generator
- internal/render - terminal rendering (tcell)
- internal/input - key handling
- internal/dockerwatch - list, events, remove containers
- internal/ai - random wandering behavior

## Game State
- GameState: maze grid, player, enemies, seed, score, tick, respawn timer
- Enemy: id, pos, containerID, alive
- ContainerIndex: map[containerID]enemyID for stable mapping

## Docker Integration
- On start: list all running containers, create one enemy per container
- Watch events: start => spawn enemy; stop/die => despawn enemy
- On enemy death: docker rm -f <containerID> for that enemy only

## Maze Generation
- Randomized backtracker/Prim to produce labyrinth
- Ensure open spawn positions; regenerate if no safe space

## Game Loop
- Each tick: read input -> move player -> move enemies -> resolve collisions -> apply docker events -> render
- Collision rule: player hits enemy => enemy dies => remove container
- Player death => respawn after respawnDelay

## Enemy AI
- Random walking, avoid walls; occasional direction change

## Rendering
- Use tcell for stable terminal control
- Draw maze, player, enemies, status bar (seed, containers, respawn)

## Concurrency
- Docker watcher goroutine pushes events to channel
- Game loop is the single state mutator

## UX & Safety
- Startup warning: game deletes running containers on enemy death
- Graceful shutdown restores terminal state
