package game

import (
	"errors"
	"time"

	"pacman-pod-killer/internal/ai"
	"pacman-pod-killer/internal/maze"
)

var ErrQuit = errors.New("quit")
var ErrGameWon = errors.New("game won")

type Position struct {
	X int
	Y int
}

// WorldPosition represents a position in the infinite world
type WorldPosition struct {
	Location string // Which location/chunk (e.g., "location1")
	LocalX   int    // Position within the chunk
	LocalY   int    // Position within the chunk
}

type Player struct {
	Position
	Alive        bool
	RespawnAt    time.Time
	RespawnDelay time.Duration
}

type Enemy struct {
	ID          string
	Position    Position
	ContainerID string
	Location    string // Which location this enemy belongs to
	Alive       bool
	// SARSA state for on-policy learning (shared Q-table, per-enemy state)
	SARSAState ai.EnemyState
}

type GameState struct {
	Maze            maze.Grid
	Player          Player
	Enemies         map[string]*Enemy
	ContainerIndex  map[string]string
	Seed            int64
	Tick            int64
	EnemySpeed      int // Enemy moves every N ticks (higher = slower enemies)
	DockerStatus    string
	DockerError     string
	CurrentLocation string             // Current location name the player is in
	ChunkManager    *maze.ChunkManager // Manages maze chunks for each location
	GameWon         bool               // True if player reached the final exit
	EnemiesKilled   int                // Total enemies killed (containers removed)
	PlayerDeaths    int                // Total player deaths
}

type EnemySnapshot struct {
	ID            string
	ContainerID   string
	ContainerName string // Container name for location matching
	Alive         bool
}
