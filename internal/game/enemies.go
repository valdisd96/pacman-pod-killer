package game

import "math/rand"

func EnemiesFromDocker(incoming []EnemySnapshot, rng *rand.Rand, state *GameState) []Enemy {
	enemies := make([]Enemy, 0, len(incoming))
	for _, info := range incoming {
		id := info.ID
		if len(id) > 12 {
			id = id[:12]
		}
		enemy := Enemy{
			ID:          id,
			ContainerID: info.ContainerID,
			Position:    state.FindOpenPosition(rng),
			Alive:       info.Alive,
		}
		enemies = append(enemies, enemy)
	}
	return enemies
}
