package ai

import (
	"pacman-pod-killer/internal/maze"
)

// CountOpenSpace returns the number of open cells (0-3) in each direction from position
// Uses ray-casting up to 3 cells to count walkable floor cells
// Returns [4]int for [up, down, left, right]
func CountOpenSpace(grid *maze.Grid, x, y int) [4]int {
	var counts [4]int
	maxDepth := 3

	// Up direction (decreasing Y)
	for i := 1; i <= maxDepth; i++ {
		nx, ny := x, y-i
		if canMoveSimple(grid, x, y, nx, ny) {
			counts[0]++
		} else {
			break
		}
	}

	// Down direction (increasing Y)
	for i := 1; i <= maxDepth; i++ {
		nx, ny := x, y+i
		if canMoveSimple(grid, x, y, nx, ny) {
			counts[1]++
		} else {
			break
		}
	}

	// Left direction (decreasing X)
	for i := 1; i <= maxDepth; i++ {
		nx, ny := x-i, y
		if canMoveSimple(grid, x, y, nx, ny) {
			counts[2]++
		} else {
			break
		}
	}

	// Right direction (increasing X)
	for i := 1; i <= maxDepth; i++ {
		nx, ny := x+i, y
		if canMoveSimple(grid, x, y, nx, ny) {
			counts[3]++
		} else {
			break
		}
	}

	return counts
}

// canMoveSimple is a simplified version that checks if destination is floor
// and within bounds. Used for open space counting.
func canMoveSimple(grid *maze.Grid, fromX, fromY, toX, toY int) bool {
	// Check bounds
	if toX < 0 || toY < 0 || toX >= grid.LogicWidth || toY >= grid.LogicHeight {
		return false
	}

	// Must be adjacent (one step)
	dx := toX - fromX
	dy := toY - fromY
	if (dx != 0 && dy != 0) || abs(dx) > 1 || abs(dy) > 1 {
		return false
	}

	// Check if destination is floor
	return grid.IsFloor(toX, toY)
}
