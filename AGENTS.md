# AGENTS.md

This repository contains a terminal Pacman game written in Go. The game uses `tcell` for rendering/input and the Docker Engine API to map running containers to enemies. These instructions are for agentic coding tools working in this repo.

## Build, Run, and Test
- Build: `go build ./...`
- Run locally: `go run ./cmd/pacman`
- Run with flags: `go run ./cmd/pacman --width 80 --height 30 --seed 123 --respawn-delay 10000 --tick-ms 80`
- macOS Docker Desktop socket example:
  `DOCKER_HOST=unix:///Users/uvauchok/.docker/run/docker.sock script -q /dev/null go run ./cmd/pacman --width 80 --height 30 --seed 123 --respawn-delay 10000 --tick-ms 80 --debug`
- Lint: no linter configured yet (use `go vet ./...` as a lightweight check)
- Unit tests: `go test ./...`
- Single test (package): `go test ./internal/maze`
- Single test (function): `go test ./internal/maze -run TestMazeGeneration`
- Format: `gofmt -w <files>` (Go formatting is mandatory)

### Common maintenance
- Tidy modules: `go mod tidy`
- Check formatting: `gofmt -w $(git ls-files "*.go")`
- Quick static check (manual): scan for TODOs and log noisy output

If you add new tools (golangci-lint, staticcheck, etc.), update this section.

## Code Style Guidelines

### Language and formatting
- Language: Go (go1.24).
- Always run `gofmt` on modified files.
- Prefer standard library first; avoid new deps unless needed.
- Keep files ASCII-only unless a file already contains Unicode.
- Use tabs for indentation in Go code (gofmt enforces this).
- Keep functions small and single-purpose; split when branching grows.

### Imports
- Standard library first, blank line, third-party, blank line, local packages.
- Avoid unused imports; keep import groups minimal.
- Avoid dot imports.
- Keep aliasing rare and only for conflict resolution.

### Types and data flow
- Favor explicit structs for shared state (`GameState`, `Enemy`, etc.).
- Use value types for immutable snapshots; use pointers for shared mutable state.
- Keep Docker types confined to `internal/dockerwatch` to avoid leaks.
- Convert Docker data to internal snapshots before handing to game logic.
- Keep render data in `render.Frame` structs to avoid coupling.

### Naming conventions
- Exported names only when used outside the package.
- Use `NewX` for constructors, `XFromY` for converters.
- Keep identifiers short but descriptive; avoid abbreviations unless common.
- Use `ID` for container IDs; avoid `Id`.
- Prefer `RespawnDelay` and `TickMillis` over cryptic names.

### Errors and control flow
- Return errors upward; do not `panic` in production paths.
- Wrap errors with context using `fmt.Errorf("...: %w", err)`.
- Keep early returns for guard clauses.
- Handle Docker errors without crashing the game loop.
- Avoid silent error swallowing unless explicitly non-critical.

### Concurrency
- Game state mutations happen in the main loop only.
- Docker watcher sends events over a channel.
- Avoid shared mutable state across goroutines.
- Ensure goroutines have a stop signal; avoid leaks.
- No blocking calls inside the tick loop.

### Rendering and input
- `tcell` screen is created once in `cmd/pacman/main.go` and passed down.
- `render.Renderer` only draws `render.Frame` values (no game state coupling).
- Input reader should not mutate state; it emits actions only.
- Rendering should not allocate large memory each frame.
- Keep status bar to a single line below the maze grid.

### Docker safety
- Each enemy maps to exactly one running container.
- On enemy death, remove only that container (`docker rm -f <id>`).
- Handle Docker errors gracefully; do not crash the game loop.
- Do not remove containers for other reasons (e.g., stop events).
- Avoid blocking event handling; buffer events where needed.

### Game loop behavior
- Tick order: apply docker events -> handle input -> move entities -> resolve collisions -> draw.
- Player respawns after a delay; respawn delay is configurable.
- Enemy AI supports two modes: random wandering or SARSA reinforcement learning.
- SARSA mode: enemies learn to chase the player using on-policy RL.
- Maze bounds are enforced before tile lookups.

### Flags and defaults
- `--width` / `--height`: maze size (default 15x10 logical cells).
- `--seed`: RNG seed (default time-based).
- `--respawn-delay`: delay in milliseconds (default 10000).
- `--tick-ms`: tick duration in milliseconds (default 80).
- `--enemy-speed`: enemy moves every N ticks (default 2, higher = slower enemies).
- `--log-file`: path to log file for game actions (default empty = disabled).
- `--debug`: enable debug logging to stdout (default false).
- `--ai-mode`: enemy AI mode, `random` or `sarsa` (default sarsa).
- `--training`: enable SARSA training mode (default true).
- `--epsilon`: SARSA exploration rate 0.0-1.0 (default 0.1).
- `--qtable`: path to Q-table file (default ~/.pacman-pod-killer/qtable.json).

### Logging and output
- Avoid stdout spam; keep warnings concise.
- Prefer stderr for errors.
- Ensure terminal state is restored even on errors.
- Use `--log-file` flag to enable file-based logging for troubleshooting.
- Log file captures: collisions, container removals, docker events, respawns.
- Logger is thread-safe and writes timestamped messages with level prefixes.

### SARSA reinforcement learning
- State representation: relative player position (clamped to [-2,+2]) + available moves.
- State space: ~400 possible states for manageable Q-table size.
- Action space: Up, Down, Left, Right (4 actions).
- Reward function: +100 catch, +1 closer, -0.5 farther, -2 wall, -0.1 step penalty.
- Hyperparameters: alpha=0.1, gamma=0.95, epsilon=0.1 (configurable).
- All enemies share a single Q-table for collective learning.
- Q-table persists to JSON file between sessions.
- SARSA update: Q(s,a) += alpha * [r + gamma * Q(s',a') - Q(s,a)].

### Maze generation
- Walls are rendered as `#` characters.
- Floor (walkable areas) are rendered as spaces for a clean look.
- Multiple maze styles available: Classic, Corridors, Rooms, Spiral, Grid.
- Each location gets a randomly chosen style (using the game's RNG seed for reproducibility).
- Connectivity is guaranteed: `EnsureConnectivity()` carves a path if none exists between entry and exit.
- Portals: green `O` for exit (next location), purple `<` for entry (previous location).

## Repository layout
- `cmd/pacman`: entrypoint and wiring
- `internal/game`: state, controller, conversion helpers
- `internal/maze`: maze generation (multiple styles, pathfinding, connectivity)
- `internal/render`: terminal rendering
- `internal/input`: input reader
- `internal/dockerwatch`: Docker client/event watcher
- `internal/ai`: enemy AI (random and SARSA reinforcement learning)
- `internal/logger`: thread-safe file-based logging

## File boundaries
- `internal/game` owns rules and state only.
- `internal/render` owns display only; it should never import `game`.
- `internal/input` reads keys and exposes actions only.
- `internal/dockerwatch` is the only package that touches Docker SDK types.
- `cmd/pacman` owns wiring, flags, and the warning banner.

## Repo-specific rules
- No Cursor rules found (.cursor/rules/ or .cursorrules).
- No Copilot instructions found (.github/copilot-instructions.md).

## Development tips
- Use `--seed` for deterministic maze reproduction.
- Start/stop containers during a session to confirm enemy sync.
- Keep the maze odd-sized for nicer labyrinths.

## Security and safety
- The game deletes running containers on enemy death; treat as destructive.
- Avoid adding any auto-delete behavior beyond explicit collisions.
- Never remove containers on startup or shutdown.

## Safety and UX notes
- The game deletes running containers on enemy death. Warn users at startup.
- Keep the tick loop responsive; avoid blocking calls.
- Ensure terminal state is restored on exit.
