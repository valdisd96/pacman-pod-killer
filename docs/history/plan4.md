# Shooting Mechanic Implementation Plan

## Goal Description
Implement a shooting mechanic where the player can fire multiple bullets using the Spacebar. Bullets travel in the direction of the player's last movement, move faster than enemies, destroy enemies on impact (one kill per bullet), and disappear when hitting walls.

## User Review Required
> [!IMPORTANT]
> **Bullet Speed**: Bullets will move every tick (speed=1). If enemies are also at max speed (1), bullets won't be faster. Default enemy speed is usually slower (>1), so this should satisfy "slightly greater".

## Proposed Changes

### Game Logic & Data Structures

#### [MODIFY] [types.go](file:///Users/uvauchok/uvauchok-projects/funny/pacman-pod-killer/internal/game/types.go)
- Add `Bullet` struct:
  ```go
  type Bullet struct {
      Position  Position
      Direction Position // dx, dy
      Alive     bool
  }
  ```
- Update `Player` struct:
  - Add `LastDirection Position` (default to right or last move)
- Update `GameState` struct:
  - Add `Bullets []Bullet`
  - Add `LastShootTick int64`
  - Add `ShootCooldown int64` (default e.g., 5 ticks)

#### [MODIFY] [input.go](file:///Users/uvauchok/uvauchok-projects/funny/pacman-pod-killer/internal/input/input.go)
- Add `ActionShoot` constant.
- Map `Spacebar` (KeyRune ' ') to `ActionShoot`.

#### [MODIFY] [controller.go](file:///Users/uvauchok/uvauchok-projects/funny/pacman-pod-killer/internal/game/controller.go)
- **Input Handling**:
  - In `handleInput`, case `ActionShoot`: call `controller.tryShoot()`.
  - In `tryMovePlayer`, update `Player.LastDirection`.
- **New Methods**:
  - `tryShoot()`: Check cooldown (`Tick - LastShootTick > Cooldown`), append new `Bullet` to `state.Bullets`.
  - `moveBullets()`: Iterate bullets. Update `X, Y` by `Direction`. Check `Maze.IsWall`. If wall, `Alive=false`.
  - `resolveBulletCollisions()`: Check intersection with enemies. If hit, `Enemy.Alive=false`, `Bullet.Alive=false`, trigger container removal, `EnemiesKilled++`.
- **Game Loop**:
  - Update `Tick()` to call `moveBullets()` and `resolveBulletCollisions()`.
  - Filter dead bullets from `state.Bullets` periodically or during movement.

### Rendering

#### [MODIFY] [internal/render/render.go](file:///Users/uvauchok/uvauchok-projects/funny/pacman-pod-killer/internal/render/render.go)
- Update `Frame` struct to include `Bullets []Position`.
- In `Draw()`: Loop through bullets and render `●` (U+25CF) in a specific color (e.g., Cyan or White).

#### [MODIFY] [internal/game/frame.go](file:///Users/uvauchok/uvauchok-projects/funny/pacman-pod-killer/internal/game/frame.go)
- Update `BuildFrame()` to populate `Frame.Bullets` from `state.Bullets`.

## Verification Plan

### Manual Verification
1.  **Start Game**: Run the game.
2.  **Movement**: Move around to set `LastDirection`.
3.  **Shooting**: Press Spacebar. Verify a `●` bullet spawns and moves in the correct direction.
4.  **Cooldown**: Spam Spacebar. Verify distinct gaps between bullets (not a continuous stream).
5.  **Walls**: Fire at a wall. Verify bullet disappears on contact.
6.  **Enemies**: Fire at an enemy. Verify enemy dies/disappears and bullet disappears.
7.  **Docker**: Verify `docker rm` logs appear (or mock output) when enemy is killed by bullet.
