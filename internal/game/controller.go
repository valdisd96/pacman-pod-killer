package game

import (
	"math/rand"
	"time"

	"pacman-pod-killer/internal/ai"
	"pacman-pod-killer/internal/dockerwatch"
	"pacman-pod-killer/internal/input"
	"pacman-pod-killer/internal/logger"
	"pacman-pod-killer/internal/render"
)

type Controller struct {
	state       *GameState
	renderer    *render.Renderer
	input       *input.Reader
	ai          *ai.Random
	enemyEvents <-chan dockerwatch.Event
	docker      *dockerwatch.Client
	rng         *rand.Rand
	log         *logger.Logger
}

func NewController(state *GameState, renderer *render.Renderer, input *input.Reader, enemyAI *ai.Random, events <-chan dockerwatch.Event, docker *dockerwatch.Client, log *logger.Logger) *Controller {
	rng := rand.New(rand.NewSource(state.Seed))
	return &Controller{state: state, renderer: renderer, input: input, ai: enemyAI, enemyEvents: events, docker: docker, rng: rng, log: log}
}

func (controller *Controller) Tick() error {
	controller.state.Tick++
	controller.applyDockerEvents()

	if err := controller.handleInput(); err != nil {
		return err
	}

	if controller.state.Player.Alive {
		controller.moveEnemies()
		controller.resolveCollisions()
	} else {
		controller.tryRespawn()
	}

	return controller.renderer.Draw(BuildFrame(controller.state))
}

func (controller *Controller) handleInput() error {
	action := controller.input.ReadAction()
	switch action {
	case input.ActionQuit:
		return ErrQuit
	case input.ActionUp:
		controller.tryMovePlayer(0, -1)
	case input.ActionDown:
		controller.tryMovePlayer(0, 1)
	case input.ActionLeft:
		controller.tryMovePlayer(-1, 0)
	case input.ActionRight:
		controller.tryMovePlayer(1, 0)
	}
	return nil
}

func (controller *Controller) tryMovePlayer(dx, dy int) {
	if !controller.state.Player.Alive {
		return
	}
	nx := controller.state.Player.X + dx
	ny := controller.state.Player.Y + dy

	// Check if movement is allowed (bounds, floor, and passage exists)
	if !controller.state.Maze.CanMove(controller.state.Player.X, controller.state.Player.Y, nx, ny) {
		return
	}

	controller.state.Player.X = nx
	controller.state.Player.Y = ny
}

func (controller *Controller) moveEnemies() {
	for _, enemy := range controller.state.Enemies {
		if !enemy.Alive {
			continue
		}
		dx, dy := controller.ai.NextStep(&controller.state.Maze, enemy.Position.X, enemy.Position.Y)
		nx := enemy.Position.X + dx
		ny := enemy.Position.Y + dy

		// Check if movement is allowed (bounds, floor, and passage exists)
		if !controller.state.Maze.CanMove(enemy.Position.X, enemy.Position.Y, nx, ny) {
			continue
		}

		enemy.Position.X = nx
		enemy.Position.Y = ny
	}
}

func (controller *Controller) resolveCollisions() {
	for _, enemy := range controller.state.Enemies {
		if !enemy.Alive {
			continue
		}
		// Collision check using logical coordinates
		if enemy.Position.X == controller.state.Player.X && enemy.Position.Y == controller.state.Player.Y {
			controller.log.Info("COLLISION: Player at (%d,%d) hit enemy %s (container=%s)",
				controller.state.Player.X, controller.state.Player.Y, enemy.ID, enemy.ContainerID)
			enemy.Alive = false
			controller.state.Player.Alive = false
			controller.state.Player.RespawnAt = time.Now().Add(controller.state.Player.RespawnDelay)
			if controller.docker != nil {
				controller.log.Info("Requesting container removal for enemy %s (container=%s)", enemy.ID, enemy.ContainerID)
				if err := controller.docker.Remove(enemy.ContainerID); err != nil {
					controller.log.Error("Failed to remove container %s: %v", enemy.ContainerID, err)
				}
			} else {
				controller.log.Info("Docker client is nil, skipping container removal")
			}
		}
	}
}

func (controller *Controller) tryRespawn() {
	if time.Now().Before(controller.state.Player.RespawnAt) {
		return
	}
	controller.state.Player.Position = controller.state.FindOpenPosition(controller.rng)
	controller.state.Player.Alive = true
	controller.log.Info("Player respawned at (%d,%d)", controller.state.Player.X, controller.state.Player.Y)
}

func (controller *Controller) applyDockerEvents() {
	if controller.enemyEvents == nil {
		return
	}
	for {
		select {
		case event := <-controller.enemyEvents:
			controller.handleDockerEvent(event)
		default:
			return
		}
	}
}

func (controller *Controller) handleDockerEvent(event dockerwatch.Event) {
	switch event.Type {
	case dockerwatch.EventStart:
		if _, exists := controller.state.ContainerIndex[event.ContainerID]; exists {
			controller.log.Debug("Ignoring EventStart for already-tracked container: %s", event.ContainerID)
			return
		}
		id := event.ContainerID
		if len(id) > 12 {
			id = id[:12]
		}
		if controller.state.DockerStatus == "docker: connected" {
			controller.state.DockerStatus = "docker: events"
		}
		enemy := Enemy{
			ID:          id,
			ContainerID: event.ContainerID,
			Position:    controller.state.FindOpenPosition(controller.rng),
			Alive:       true,
		}
		controller.state.Enemies[enemy.ID] = &enemy
		controller.state.ContainerIndex[enemy.ContainerID] = enemy.ID
		controller.log.Info("Enemy spawned: id=%s container=%s at (%d,%d)", enemy.ID, enemy.ContainerID, enemy.Position.X, enemy.Position.Y)
	case dockerwatch.EventStop:
		if enemyID, exists := controller.state.ContainerIndex[event.ContainerID]; exists {
			controller.log.Info("Enemy removed via EventStop: id=%s container=%s", enemyID, event.ContainerID)
			if enemy, ok := controller.state.Enemies[enemyID]; ok {
				enemy.Alive = false
			}
			delete(controller.state.Enemies, enemyID)
			delete(controller.state.ContainerIndex, event.ContainerID)
		} else {
			controller.log.Debug("EventStop for unknown container: %s", event.ContainerID)
		}
	}
}
