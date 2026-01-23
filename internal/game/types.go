package game

import (
	"errors"
	"time"

	"pacman-pod-killer/internal/maze"
)

var ErrQuit = errors.New("quit")

type Position struct {
	X int
	Y int
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
	Alive       bool
}

type GameState struct {
	Maze           maze.Grid
	Player         Player
	Enemies        map[string]*Enemy
	ContainerIndex map[string]string
	Seed           int64
	Tick           int64
	DockerStatus   string
	DockerError    string
}

type EnemySnapshot struct {
	ID          string
	ContainerID string
	Alive       bool
}
