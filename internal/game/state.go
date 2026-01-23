package game

import (
	"math/rand"
	"time"

	"pacman-pod-killer/internal/maze"
)

func NewState(grid maze.Grid, seed int64, respawnDelay time.Duration) *GameState {
	player := Player{
		Position:     Position{X: 0, Y: 0}, // Start at logical (0, 0)
		Alive:        true,
		RespawnDelay: respawnDelay,
	}

	return &GameState{
		Maze:           grid,
		Player:         player,
		Enemies:        map[string]*Enemy{},
		ContainerIndex: map[string]string{},
		Seed:           seed,
		DockerStatus:   "docker: unknown",
		DockerError:    "",
	}
}

// FindOpenPosition finds a random open logical position in the maze
func (state *GameState) FindOpenPosition(rng *rand.Rand) Position {
	for attempts := 0; attempts < 2000; attempts++ {
		// Generate random logical coordinates
		lx := rng.Intn(state.Maze.LogicWidth)
		ly := rng.Intn(state.Maze.LogicHeight)

		// Check if it's a floor tile
		if !state.Maze.IsFloor(lx, ly) {
			continue
		}

		// Can't be player position
		if state.Player.Alive && state.Player.X == lx && state.Player.Y == ly {
			continue
		}

		// Can't be occupied by another enemy
		occupied := false
		for _, enemy := range state.Enemies {
			if enemy.Alive && enemy.Position.X == lx && enemy.Position.Y == ly {
				occupied = true
				break
			}
		}
		if !occupied {
			return Position{X: lx, Y: ly}
		}
	}
	return Position{X: 0, Y: 0} // Fallback to logical (0, 0)
}

func (state *GameState) SyncEnemies(incoming []Enemy) {
	state.Enemies = map[string]*Enemy{}
	state.ContainerIndex = map[string]string{}
	for _, enemy := range incoming {
		copy := enemy
		state.Enemies[enemy.ID] = &copy
		state.ContainerIndex[enemy.ContainerID] = enemy.ID
	}
}
