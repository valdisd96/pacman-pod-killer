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

// AIMode represents the enemy AI algorithm to use
type AIMode string

const (
	AIModeRandom AIMode = "random"
	AIModeSARSA  AIMode = "sarsa"
)

type Controller struct {
	state       *GameState
	renderer    *render.Renderer
	input       *input.Reader
	randomAI    *ai.Random
	sarsaAI     *ai.SARSA
	aiMode      AIMode
	enemyEvents <-chan dockerwatch.Event
	docker      *dockerwatch.Client
	rng         *rand.Rand
	log         *logger.Logger
}

// ControllerConfig holds configuration for the game controller
type ControllerConfig struct {
	State       *GameState
	Renderer    *render.Renderer
	Input       *input.Reader
	RandomAI    *ai.Random
	SarsaAI     *ai.SARSA
	AIMode      AIMode
	EnemyEvents <-chan dockerwatch.Event
	Docker      *dockerwatch.Client
	Log         *logger.Logger
}

// NewController creates a new game controller
func NewController(cfg ControllerConfig) *Controller {
	rng := rand.New(rand.NewSource(cfg.State.Seed))
	aiMode := cfg.AIMode
	if aiMode == "" {
		aiMode = AIModeSARSA // Default to SARSA
	}
	return &Controller{
		state:       cfg.State,
		renderer:    cfg.Renderer,
		input:       cfg.Input,
		randomAI:    cfg.RandomAI,
		sarsaAI:     cfg.SarsaAI,
		aiMode:      aiMode,
		enemyEvents: cfg.EnemyEvents,
		docker:      cfg.Docker,
		rng:         rng,
		log:         cfg.Log,
	}
}

// SaveQTable saves the SARSA Q-table to disk (if using SARSA mode)
func (controller *Controller) SaveQTable(path string) error {
	if controller.sarsaAI == nil {
		return nil
	}
	return controller.sarsaAI.QTable().Save(path)
}

func (controller *Controller) Tick() error {
	controller.state.Tick++
	controller.applyDockerEvents()

	if err := controller.handleInput(); err != nil {
		return err
	}

	// Move bullets every tick (faster than enemies)
	controller.moveBullets()
	controller.resolveBulletCollisions()

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
	case input.ActionShoot:
		controller.tryShoot()
	}
	return nil
}

func (controller *Controller) tryShoot() {
	if !controller.state.Player.Alive {
		return
	}

	// Check cooldown
	if controller.state.Tick-controller.state.LastShootTick < controller.state.ShootCooldown {
		return
	}

	// Ensure player has a direction to shoot
	dir := controller.state.Player.LastDirection
	if dir.X == 0 && dir.Y == 0 {
		// Default to right if no direction set yet
		dir = Position{X: 1, Y: 0}
	}

	// Create bullet at player's position
	bullet := Bullet{
		Position:  controller.state.Player.Position,
		Direction: dir,
		Alive:     true,
	}

	controller.state.Bullets = append(controller.state.Bullets, bullet)
	controller.state.LastShootTick = controller.state.Tick
	controller.log.Info("Player fired bullet at (%d,%d) direction (%d,%d)",
		bullet.Position.X, bullet.Position.Y, dir.X, dir.Y)
}

func (controller *Controller) moveBullets() {
	for i := range controller.state.Bullets {
		bullet := &controller.state.Bullets[i]
		if !bullet.Alive {
			continue
		}

		nx := bullet.Position.X + bullet.Direction.X
		ny := bullet.Position.Y + bullet.Direction.Y

		// Check if movement is allowed (bounds, floor, and passage exists)
		// This uses CanMove which checks for walls between cells
		if !controller.state.Maze.CanMove(bullet.Position.X, bullet.Position.Y, nx, ny) {
			bullet.Alive = false
			controller.log.Info("Bullet at (%d,%d) hit wall at (%d,%d)", bullet.Position.X, bullet.Position.Y, nx, ny)
			continue
		}

		// Move bullet
		bullet.Position.X = nx
		bullet.Position.Y = ny
	}

	// Remove dead bullets
	controller.filterDeadBullets()
}

func (controller *Controller) filterDeadBullets() {
	alive := controller.state.Bullets[:0]
	for _, bullet := range controller.state.Bullets {
		if bullet.Alive {
			alive = append(alive, bullet)
		}
	}
	controller.state.Bullets = alive
}

func (controller *Controller) resolveBulletCollisions() {
	for i := range controller.state.Bullets {
		bullet := &controller.state.Bullets[i]
		if !bullet.Alive {
			continue
		}

		for _, enemy := range controller.state.Enemies {
			if !enemy.Alive {
				continue
			}

			// Only check enemies in the current location
			if controller.state.ChunkManager != nil && enemy.Location != controller.state.CurrentLocation {
				continue
			}

			// Check if bullet hits enemy
			if bullet.Position.X == enemy.Position.X && bullet.Position.Y == enemy.Position.Y {
				controller.log.Info("BULLET HIT: Bullet at (%d,%d) hit enemy %s (container=%s)",
					bullet.Position.X, bullet.Position.Y, enemy.ID, enemy.ContainerID)

				bullet.Alive = false
				enemy.Alive = false
				controller.state.EnemiesKilled++

				if controller.docker != nil {
					controller.log.Info("Requesting container removal for enemy %s (container=%s)", enemy.ID, enemy.ContainerID)
					if err := controller.docker.Remove(enemy.ContainerID); err != nil {
						controller.log.Error("Failed to remove container %s: %v", enemy.ContainerID, err)
					}
				}

				// One bullet kills one enemy, then bullet dies
				break
			}
		}
	}
}

func (controller *Controller) tryMovePlayer(dx, dy int) {
	if !controller.state.Player.Alive {
		return
	}

	// Check for game won state
	if controller.state.GameWon {
		return
	}

	nx := controller.state.Player.X + dx
	ny := controller.state.Player.Y + dy

	// Check for portal transitions (location changes)
	if controller.state.ChunkManager != nil {
		// Check if player is at WIN portal and moving right (final location)
		winPortal := controller.state.ChunkManager.GetWinPortal(controller.state.CurrentLocation)
		if winPortal != nil && dx > 0 &&
			controller.state.Player.X == winPortal.X && controller.state.Player.Y == winPortal.Y {
			controller.log.Info("Player entering WIN portal in %s - GAME WON!", controller.state.CurrentLocation)
			controller.state.GameWon = true
			return
		}

		// Check if player is at exit portal and moving right
		exitPortal := controller.state.ChunkManager.GetExitPortal(controller.state.CurrentLocation)
		if exitPortal != nil && !exitPortal.IsWinExit && dx > 0 &&
			controller.state.Player.X == exitPortal.X && controller.state.Player.Y == exitPortal.Y {
			nextLoc := controller.state.ChunkManager.NextLocation(controller.state.CurrentLocation)
			if nextLoc != nil {
				controller.log.Info("Player entering exit portal: %s -> %s", controller.state.CurrentLocation, nextLoc.Name)
				if controller.state.SwitchLocation(nextLoc.Name, controller.rng) {
					// Find entry portal in the new location
					entryPortal := controller.state.ChunkManager.GetEntryPortal(nextLoc.Name)
					if entryPortal != nil {
						controller.state.Player.X = entryPortal.X
						controller.state.Player.Y = entryPortal.Y
					} else {
						// Fallback: enter at left edge, keep Y position
						controller.state.Player.X = 0
					}
					return
				}
			}
		}

		// Check if player is at entry portal and moving left
		entryPortal := controller.state.ChunkManager.GetEntryPortal(controller.state.CurrentLocation)
		if entryPortal != nil && dx < 0 &&
			controller.state.Player.X == entryPortal.X && controller.state.Player.Y == entryPortal.Y {
			prevLoc := controller.state.ChunkManager.PrevLocation(controller.state.CurrentLocation)
			if prevLoc != nil {
				controller.log.Info("Player entering entry portal: %s -> %s", controller.state.CurrentLocation, prevLoc.Name)
				if controller.state.SwitchLocation(prevLoc.Name, controller.rng) {
					// Find exit portal in the previous location
					prevExitPortal := controller.state.ChunkManager.GetExitPortal(prevLoc.Name)
					if prevExitPortal != nil {
						controller.state.Player.X = prevExitPortal.X
						controller.state.Player.Y = prevExitPortal.Y
					} else {
						// Fallback: enter at right edge, keep Y position
						controller.state.Player.X = controller.state.Maze.LogicWidth - 1
					}
					return
				}
			}
		}
	}

	// Check if movement is allowed (bounds, floor, and passage exists)
	if !controller.state.Maze.CanMove(controller.state.Player.X, controller.state.Player.Y, nx, ny) {
		return
	}

	controller.state.Player.X = nx
	controller.state.Player.Y = ny

	// Track the last direction for shooting
	controller.state.Player.LastDirection = Position{X: dx, Y: dy}
}

func (controller *Controller) moveEnemies() {
	// Only move enemies every N ticks based on EnemySpeed
	// EnemySpeed=1 means move every tick, EnemySpeed=2 means every 2nd tick, etc.
	if controller.state.EnemySpeed > 1 && controller.state.Tick%int64(controller.state.EnemySpeed) != 0 {
		return
	}

	for _, enemy := range controller.state.Enemies {
		if !enemy.Alive {
			continue
		}

		// Only move enemies in the current location
		if controller.state.ChunkManager != nil && enemy.Location != controller.state.CurrentLocation {
			continue
		}

		var dx, dy int
		if controller.aiMode == AIModeSARSA && controller.sarsaAI != nil {
			// Use SARSA with learning
			dx, dy = controller.sarsaAI.Step(
				enemy.Position.X, enemy.Position.Y,
				controller.state.Player.X, controller.state.Player.Y,
				&controller.state.Maze,
				&enemy.SARSAState,
				false, // not caught yet
			)
		} else {
			// Use random AI
			dx, dy = controller.randomAI.NextStep(&controller.state.Maze, enemy.Position.X, enemy.Position.Y)
		}

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

		// Only check collisions with enemies in the current location
		if controller.state.ChunkManager != nil && enemy.Location != controller.state.CurrentLocation {
			continue
		}

		// Collision check using logical coordinates
		if enemy.Position.X == controller.state.Player.X && enemy.Position.Y == controller.state.Player.Y {
			controller.log.Info("COLLISION: Player at (%d,%d) hit enemy %s (container=%s) in %s",
				controller.state.Player.X, controller.state.Player.Y, enemy.ID, enemy.ContainerID, controller.state.CurrentLocation)

			// Give SARSA reward for catching the player
			if controller.aiMode == AIModeSARSA && controller.sarsaAI != nil {
				controller.sarsaAI.HandlePlayerCaught(
					&enemy.SARSAState,
					&controller.state.Maze,
					enemy.Position.X, enemy.Position.Y,
					controller.state.Player.X, controller.state.Player.Y,
				)
			}

			enemy.Alive = false
			controller.state.Player.Alive = false
			controller.state.Player.RespawnAt = time.Now().Add(controller.state.Player.RespawnDelay)
			controller.state.EnemiesKilled++
			controller.state.PlayerDeaths++
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

		// Determine location based on container name
		location := DetermineLocation(event.ContainerName, controller.state)

		enemy := Enemy{
			ID:          id,
			ContainerID: event.ContainerID,
			Location:    location,
			Position:    controller.state.FindOpenPosition(controller.rng),
			Alive:       true,
		}
		controller.state.Enemies[enemy.ID] = &enemy
		controller.state.ContainerIndex[enemy.ContainerID] = enemy.ID
		controller.log.Info("Enemy spawned: id=%s container=%s name=%s location=%s at (%d,%d)",
			enemy.ID, enemy.ContainerID, event.ContainerName, location, enemy.Position.X, enemy.Position.Y)
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
