package game

import "pacman-pod-killer/internal/render"

func BuildFrame(state *GameState) render.Frame {
	frame := render.Frame{
		Maze:       state.Maze,
		PlayerX:    state.Player.X,
		PlayerY:    state.Player.Y,
		PlayerLive: state.Player.Alive,
		Seed:       state.Seed,
		Tick:       state.Tick,
		DockerInfo: state.DockerStatus,
		DockerErr:  state.DockerError,
	}
	for _, enemy := range state.Enemies {
		if !enemy.Alive {
			continue
		}
		frame.Enemies = append(frame.Enemies, render.Position{X: enemy.Position.X, Y: enemy.Position.Y})
	}
	return frame
}
