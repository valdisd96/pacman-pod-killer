package game

import (
	"pacman-pod-killer/internal/ai"
	"pacman-pod-killer/internal/render"
)

func BuildFrame(state *GameState, metrics *ai.Metrics) render.Frame {
	frame := render.Frame{
		Maze:            state.Maze,
		PlayerX:         state.Player.X,
		PlayerY:         state.Player.Y,
		PlayerLive:      state.Player.Alive,
		Seed:            state.Seed,
		Tick:            state.Tick,
		DockerInfo:      state.DockerStatus,
		DockerErr:       state.DockerError,
		CurrentLocation: state.CurrentLocation,
		GameWon:         state.GameWon,
		EnemiesKilled:   state.EnemiesKilled,
		PlayerDeaths:    state.PlayerDeaths,
	}

	// Add AI metrics if available
	if metrics != nil {
		frame.AIMetricsDisplay = metrics.GetDisplayString()
	}

	// Set total locations and portals if chunk manager is active
	if state.ChunkManager != nil {
		frame.TotalLocations = state.ChunkManager.LocationCount()

		// Add portals for the current location
		portals := state.ChunkManager.GetPortals(state.CurrentLocation)
		for _, p := range portals {
			frame.Portals = append(frame.Portals, render.PortalInfo{
				X:         p.X,
				Y:         p.Y,
				IsExit:    p.IsExit,
				TargetLoc: p.TargetLoc,
			})
		}
	}

	// Only include enemies from the current location
	for _, enemy := range state.Enemies {
		if !enemy.Alive {
			continue
		}
		// Filter by location if chunk manager is active
		if state.ChunkManager != nil && enemy.Location != state.CurrentLocation {
			continue
		}
		frame.Enemies = append(frame.Enemies, render.Position{X: enemy.Position.X, Y: enemy.Position.Y})
	}

	// Add bullets
	for _, bullet := range state.Bullets {
		if bullet.Alive {
			frame.Bullets = append(frame.Bullets, render.Position{X: bullet.Position.X, Y: bullet.Position.Y})
		}
	}

	return frame
}
