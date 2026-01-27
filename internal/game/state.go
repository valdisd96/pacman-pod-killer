package game

import (
	"math/rand"
	"time"

	"pacman-pod-killer/internal/maze"
)

// StateConfig holds configuration for creating a new game state
type StateConfig struct {
	Seed         int64
	RespawnDelay time.Duration
	EnemySpeed   int
	Locations    []string // Ordered list of location names (e.g., ["location1", "location2", "location3"])
	LogicWidth   int
	LogicHeight  int
}

func NewState(grid maze.Grid, seed int64, respawnDelay time.Duration, enemySpeed int) *GameState {
	return NewStateWithConfig(StateConfig{
		Seed:         seed,
		RespawnDelay: respawnDelay,
		EnemySpeed:   enemySpeed,
		Locations:    nil, // No locations = single maze mode
		LogicWidth:   grid.LogicWidth,
		LogicHeight:  grid.LogicHeight,
	}, &grid)
}

// NewStateWithConfig creates a new game state with location support
func NewStateWithConfig(cfg StateConfig, initialGrid *maze.Grid) *GameState {
	player := Player{
		Position:      Position{X: 0, Y: 0}, // Start at logical (0, 0)
		Alive:         true,
		RespawnDelay:  cfg.RespawnDelay,
		LastDirection: Position{X: 1, Y: 0}, // Default to right
	}

	if cfg.EnemySpeed < 1 {
		cfg.EnemySpeed = 1
	}

	state := &GameState{
		Player:         player,
		Enemies:        map[string]*Enemy{},
		ContainerIndex: map[string]string{},
		Seed:           cfg.Seed,
		EnemySpeed:     cfg.EnemySpeed,
		DockerStatus:   "docker: unknown",
		DockerError:    "",
		ShootCooldown:  5, // Cooldown between shots (in ticks)
	}

	// Initialize chunk manager if locations are specified
	if len(cfg.Locations) > 0 {
		rng := rand.New(rand.NewSource(cfg.Seed))
		state.ChunkManager = maze.NewChunkManager(cfg.Locations, cfg.LogicWidth, cfg.LogicHeight, rng)
		state.CurrentLocation = state.ChunkManager.FirstLocation()
		// Generate the first location's maze
		state.Maze = *state.ChunkManager.GetOrGenerate(state.CurrentLocation, rng)
	} else if initialGrid != nil {
		// Single maze mode (backwards compatible)
		state.Maze = *initialGrid
		state.CurrentLocation = ""
	}

	return state
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

// EnemiesInCurrentLocation returns enemies that belong to the current location
func (state *GameState) EnemiesInCurrentLocation() []*Enemy {
	var enemies []*Enemy
	for _, enemy := range state.Enemies {
		if enemy.Alive && enemy.Location == state.CurrentLocation {
			enemies = append(enemies, enemy)
		}
	}
	return enemies
}

// SwitchLocation changes the current location and updates the maze
func (state *GameState) SwitchLocation(newLocation string, rng *rand.Rand) bool {
	if state.ChunkManager == nil {
		return false
	}

	loc := state.ChunkManager.GetLocation(newLocation)
	if loc == nil {
		return false
	}

	state.CurrentLocation = newLocation
	state.Maze = *state.ChunkManager.GetOrGenerate(newLocation, rng)

	// Note: GameWon is set when player enters the WIN portal, not just entering the final location
	return true
}
