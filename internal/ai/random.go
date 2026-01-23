package ai

import (
	"math/rand"

	"pacman-pod-killer/internal/maze"
)

type Random struct {
	rng *rand.Rand
}

func NewRandom() *Random {
	return &Random{rng: rand.New(rand.NewSource(rand.Int63()))}
}

// NextStep returns the next movement delta in logical coordinates
func (ai *Random) NextStep(grid *maze.Grid, x, y int) (int, int) {
	dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	ai.rng.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
	for _, dir := range dirs {
		nx := x + dir[0]
		ny := y + dir[1]
		// Check if movement is allowed (bounds, floor, and passage exists)
		if !grid.CanMove(x, y, nx, ny) {
			continue
		}
		return dir[0], dir[1]
	}
	return 0, 0
}
