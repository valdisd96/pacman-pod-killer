package maze

import "math/rand"

const (
	Wall  = '#'
	Floor = '.'

	// Grid scale constants
	WallThickness = 2                             // Wall thickness in characters
	CorridorWidth = 3                             // Corridor width in characters
	CellSize      = WallThickness + CorridorWidth // Total cell size (5)
	EntitySize    = 3                             // Entity size in characters
)

type Grid struct {
	Width       int // Screen width in characters
	Height      int // Screen height in characters
	LogicWidth  int // Logical width (number of traversable cells horizontally)
	LogicHeight int // Logical height (number of traversable cells vertically)
	Cells       [][]rune
}

// LogicToScreen converts logical coordinates to screen coordinates (top-left of the corridor)
func LogicToScreen(lx, ly int) (int, int) {
	sx := WallThickness + lx*CellSize
	sy := WallThickness + ly*CellSize
	return sx, sy
}

// ScreenToLogic converts screen coordinates to logical coordinates
func ScreenToLogic(sx, sy int) (int, int) {
	lx := (sx - WallThickness) / CellSize
	ly := (sy - WallThickness) / CellSize
	return lx, ly
}

// IsFloor checks if a logical position is a floor tile
func (g *Grid) IsFloor(lx, ly int) bool {
	if lx < 0 || ly < 0 || lx >= g.LogicWidth || ly >= g.LogicHeight {
		return false
	}
	sx, sy := LogicToScreen(lx, ly)
	if sy < 0 || sy >= g.Height || sx < 0 || sx >= g.Width {
		return false
	}
	return g.Cells[sy][sx] == Floor
}

// CanMove checks if movement from one logical cell to an adjacent logical cell is allowed.
// This verifies that there is a passage (no wall) between the two cells.
func (g *Grid) CanMove(fromX, fromY, toX, toY int) bool {
	// Check bounds for both positions
	if fromX < 0 || fromY < 0 || fromX >= g.LogicWidth || fromY >= g.LogicHeight {
		return false
	}
	if toX < 0 || toY < 0 || toX >= g.LogicWidth || toY >= g.LogicHeight {
		return false
	}

	// Must be adjacent (one step in cardinal direction)
	dx := toX - fromX
	dy := toY - fromY
	if (dx != 0 && dy != 0) || (dx == 0 && dy == 0) {
		return false // Diagonal or no movement
	}
	if dx < -1 || dx > 1 || dy < -1 || dy > 1 {
		return false // More than one step
	}

	// Check that destination is a floor
	if !g.IsFloor(toX, toY) {
		return false
	}

	// Check that the passage between cells is clear (no wall in between)
	// The passage is in the wall area between the two corridor cells
	fromSX, fromSY := LogicToScreen(fromX, fromY)
	toSX, toSY := LogicToScreen(toX, toY)

	// Check the middle point of the passage
	// For horizontal movement: check the column between the two cells
	// For vertical movement: check the row between the two cells
	if dx != 0 {
		// Horizontal movement - check the wall column between cells
		var checkX int
		if dx > 0 {
			checkX = fromSX + CorridorWidth // Right edge of from cell
		} else {
			checkX = toSX + CorridorWidth // Right edge of to cell
		}
		// Check the middle of the corridor height
		checkY := fromSY + CorridorWidth/2
		if checkX < 0 || checkX >= g.Width || checkY < 0 || checkY >= g.Height {
			return false
		}
		return g.Cells[checkY][checkX] == Floor
	} else {
		// Vertical movement - check the wall row between cells
		var checkY int
		if dy > 0 {
			checkY = fromSY + CorridorWidth // Bottom edge of from cell
		} else {
			checkY = toSY + CorridorWidth // Bottom edge of to cell
		}
		// Check the middle of the corridor width
		checkX := fromSX + CorridorWidth/2
		if checkX < 0 || checkX >= g.Width || checkY < 0 || checkY >= g.Height {
			return false
		}
		return g.Cells[checkY][checkX] == Floor
	}
}

// Generate creates a maze with thick walls and wide corridors
// logicWidth and logicHeight specify the number of logical cells (corridors)
func Generate(logicWidth, logicHeight int, rng *rand.Rand) Grid {
	if logicWidth < 3 {
		logicWidth = 3
	}
	if logicHeight < 3 {
		logicHeight = 3
	}

	// Calculate screen dimensions
	// Screen width = wall + (corridor + wall) * logicWidth + wall
	// Simplified: 2*wall + logicWidth * cellSize - wall = wall + logicWidth * cellSize
	// Actually: left_wall + logicWidth * (corridor + wall) = wall + logicWidth * cellSize
	// But we need right wall too, so:
	// width = wall + corridor + wall + corridor + ... + wall
	// For N logical cells: wall + N * (corridor + wall) = wall + N * cellSize
	// But that gives wall thickness of 1 on right, let's recalc:
	// We want: [WALL][CORRIDOR][WALL][CORRIDOR][WALL]...
	// For N corridors: (N+1)*wall + N*corridor = (N+1)*2 + N*3 = 2N + 2 + 3N = 5N + 2
	screenWidth := logicWidth*CellSize + WallThickness
	screenHeight := logicHeight*CellSize + WallThickness

	cells := make([][]rune, screenHeight)
	for y := 0; y < screenHeight; y++ {
		row := make([]rune, screenWidth)
		for x := 0; x < screenWidth; x++ {
			row[x] = Wall
		}
		cells[y] = row
	}

	// Helper to carve a corridor cell (3x3 floor area)
	carve := func(lx, ly int) {
		sx, sy := LogicToScreen(lx, ly)
		for dy := 0; dy < CorridorWidth; dy++ {
			for dx := 0; dx < CorridorWidth; dx++ {
				if sy+dy < screenHeight && sx+dx < screenWidth {
					cells[sy+dy][sx+dx] = Floor
				}
			}
		}
	}

	// Helper to carve a passage between two adjacent logical cells
	carvePassage := func(lx1, ly1, lx2, ly2 int) {
		sx1, sy1 := LogicToScreen(lx1, ly1)
		sx2, sy2 := LogicToScreen(lx2, ly2)

		if lx1 == lx2 {
			// Vertical passage
			minY := sy1
			if sy2 < sy1 {
				minY = sy2
			}
			// Carve from the end of one corridor to the start of the next
			for y := minY; y < minY+CellSize+CorridorWidth; y++ {
				for dx := 0; dx < CorridorWidth; dx++ {
					if y < screenHeight && sx1+dx < screenWidth {
						cells[y][sx1+dx] = Floor
					}
				}
			}
		} else {
			// Horizontal passage
			minX := sx1
			if sx2 < sx1 {
				minX = sx2
			}
			// Carve from the end of one corridor to the start of the next
			for x := minX; x < minX+CellSize+CorridorWidth; x++ {
				for dy := 0; dy < CorridorWidth; dy++ {
					if sy1+dy < screenHeight && x < screenWidth {
						cells[sy1+dy][x] = Floor
					}
				}
			}
		}
	}

	// Track visited logical cells
	visited := make([][]bool, logicHeight)
	for y := 0; y < logicHeight; y++ {
		visited[y] = make([]bool, logicWidth)
	}

	// Start at logical (0, 0)
	carve(0, 0)
	visited[0][0] = true

	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	stack := [][2]int{{0, 0}}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		cx, cy := current[0], current[1]
		rng.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })

		carved := false
		for _, dir := range dirs {
			nx, ny := cx+dir[0], cy+dir[1]
			if nx < 0 || ny < 0 || nx >= logicWidth || ny >= logicHeight {
				continue
			}
			if visited[ny][nx] {
				continue
			}

			carve(nx, ny)
			carvePassage(cx, cy, nx, ny)
			visited[ny][nx] = true
			stack = append(stack, [2]int{nx, ny})
			carved = true
			break
		}

		if !carved {
			stack = stack[:len(stack)-1]
		}
	}

	return Grid{
		Width:       screenWidth,
		Height:      screenHeight,
		LogicWidth:  logicWidth,
		LogicHeight: logicHeight,
		Cells:       cells,
	}
}
