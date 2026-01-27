# Endless Labyrinth with Wrap-Around Teleportation

Enable player to run away from enemies by exiting map boundary and appearing on the opposite side in a newly generated area.

## Approved Design

- **Enemy chase**: Infinite
- **Movement style**: Wrap-around teleportation (exit top → appear at bottom in new chunk)
- **Visible area**: Fixed (e.g., 44x12 from `--width`/`--height`)
- **Chunk = visible area**: Each chunk is one screen-sized maze

---

## Proposed Changes

### Component 1: Chunk Manager

#### [NEW] chunk.go (`internal/maze/chunk.go`)

- `ChunkCoord` struct: `(ChunkX, ChunkY int)` for infinite grid position
- `ChunkManager` struct: stores generated mazes in `map[ChunkCoord]*Grid`
- `GetOrGenerate(cx, cy)` - returns existing or generates new chunk
- Edge connectivity: passages at borders align between chunks

---

### Component 2: Straighter Corridors

#### [MODIFY] maze.go (`internal/maze/maze.go`)

Update `Generate()` to favor straight corridors:
- 70% chance to continue in same direction
- 30% chance to pick random direction
- Add `GenerateWithEdges()` for edge-connected chunks

---

### Component 3: World Position Tracking

#### [MODIFY] types.go (`internal/game/types.go`)

```go
type WorldPosition struct {
    ChunkX, ChunkY int  // Which chunk
    LocalX, LocalY int  // Position within chunk
}
```

#### [MODIFY] state.go (`internal/game/state.go`)

- Add `ChunkManager` to `GameState`
- Track `CurrentChunk` being displayed

---

### Component 4: Boundary Crossing

#### [MODIFY] controller.go (`internal/game/controller.go`)

Update `tryMovePlayer()`:
- Moving past right edge: `ChunkX += 1`, `LocalX = 0`
- Moving past top edge: `ChunkY -= 1`, `LocalY = LogicHeight - 1`
- Load/generate new chunk, update displayed maze

---

### Component 5: Status Display

#### [MODIFY] render.go (`internal/render/render.go`)

Show chunk in status bar: `chunk:(0,1)`

---

## Verification

```bash
go test ./... -v

DOCKER_HOST=unix:///Users/uvauchok/.docker/run/docker.sock script -q /dev/null go run ./cmd/pacman --width 44 --height 12 --seed 123 --respawn-delay 10000 --epsilon 0.3 --tick-ms 80 --log-file game.log --enemy-speed 4
```

1. Walk to edge → verify teleport to new area
2. Verify chunk coordinates in status bar
3. Verify straighter corridors
4. Verify enemies follow through boundaries
