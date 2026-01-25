package game

import (
	"math/rand"
	"strings"
)

func EnemiesFromDocker(incoming []EnemySnapshot, rng *rand.Rand, state *GameState) []Enemy {
	enemies := make([]Enemy, 0, len(incoming))
	for _, info := range incoming {
		id := info.ID
		if len(id) > 12 {
			id = id[:12]
		}

		// Determine location based on container name pattern
		location := DetermineLocation(info.ContainerName, state)

		enemy := Enemy{
			ID:          id,
			ContainerID: info.ContainerID,
			Location:    location,
			Position:    state.FindOpenPosition(rng),
			Alive:       info.Alive,
		}
		enemies = append(enemies, enemy)
	}
	return enemies
}

// DetermineLocation finds which location an enemy belongs to based on container name
// Container names are matched as prefixes: "location1" matches "location1", "location1-1", "location1-foo", etc.
func DetermineLocation(containerName string, state *GameState) string {
	if state.ChunkManager == nil {
		return "" // No locations configured
	}

	// Try to match container name against location patterns
	locations := state.ChunkManager.Locations()
	for _, loc := range locations {
		// Match if container name starts with location name
		// e.g., "location1" matches "location1", "location1-1", "location1-foo"
		if strings.HasPrefix(containerName, loc) {
			return loc
		}
	}

	// Default to first location if no match
	if len(locations) > 0 {
		return locations[0]
	}
	return ""
}
