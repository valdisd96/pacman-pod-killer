package maze

import "math/rand"

const (
	Wall  = '#'
	Floor = ' '

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
	return GenerateWithStraightBias(logicWidth, logicHeight, rng, 0.0)
}

// GenerateWithStraightBias creates a maze favoring straighter corridors
// straightBias is the probability (0.0-1.0) to continue in the same direction
// A bias of 0.7 means 70% chance to continue straight, 30% to turn
func GenerateWithStraightBias(logicWidth, logicHeight int, rng *rand.Rand, straightBias float64) Grid {
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
	lastDir := [2]int{1, 0} // Track last direction for straight bias

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		cx, cy := current[0], current[1]

		// Apply straight bias: if we have a preferred direction, try it first with given probability
		if straightBias > 0 && rng.Float64() < straightBias {
			// Try to continue in the same direction first
			nx, ny := cx+lastDir[0], cy+lastDir[1]
			if nx >= 0 && ny >= 0 && nx < logicWidth && ny < logicHeight && !visited[ny][nx] {
				carve(nx, ny)
				carvePassage(cx, cy, nx, ny)
				visited[ny][nx] = true
				stack = append(stack, [2]int{nx, ny})
				continue
			}
		}

		// Standard random direction selection
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
			lastDir = dir // Update last direction
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

// GenerateRooms creates a maze with open rooms connected by passages
func GenerateRooms(logicWidth, logicHeight int, rng *rand.Rand) Grid {
	if logicWidth < 3 {
		logicWidth = 3
	}
	if logicHeight < 3 {
		logicHeight = 3
	}

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

	// Helper to carve a corridor cell
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
			for x := minX; x < minX+CellSize+CorridorWidth; x++ {
				for dy := 0; dy < CorridorWidth; dy++ {
					if sy1+dy < screenHeight && x < screenWidth {
						cells[sy1+dy][x] = Floor
					}
				}
			}
		}
	}

	// Create rooms: carve out rectangular areas
	// Room size: 2x2 to 3x3 logical cells
	roomCount := (logicWidth * logicHeight) / 12
	if roomCount < 2 {
		roomCount = 2
	}

	type room struct {
		x, y, w, h int
	}
	rooms := make([]room, 0, roomCount)

	for i := 0; i < roomCount; i++ {
		// Random room size and position
		w := 2 + rng.Intn(2) // 2-3 cells wide
		h := 2 + rng.Intn(2) // 2-3 cells tall
		if w > logicWidth-1 {
			w = logicWidth - 1
		}
		if h > logicHeight-1 {
			h = logicHeight - 1
		}

		x := rng.Intn(logicWidth - w)
		y := rng.Intn(logicHeight - h)

		// Carve the room
		for ly := y; ly < y+h; ly++ {
			for lx := x; lx < x+w; lx++ {
				carve(lx, ly)
				// Connect adjacent cells within the room
				if lx > x {
					carvePassage(lx-1, ly, lx, ly)
				}
				if ly > y {
					carvePassage(lx, ly-1, lx, ly)
				}
			}
		}

		rooms = append(rooms, room{x, y, w, h})
	}

	// Connect rooms with corridors using minimum spanning tree approach
	// First, ensure all cells are connected using DFS from (0,0)
	visited := make([][]bool, logicHeight)
	for y := 0; y < logicHeight; y++ {
		visited[y] = make([]bool, logicWidth)
	}

	// Mark room cells as visited
	for _, r := range rooms {
		for ly := r.y; ly < r.y+r.h; ly++ {
			for lx := r.x; lx < r.x+r.w; lx++ {
				visited[ly][lx] = true
			}
		}
	}

	// Connect rooms by drawing corridors between their centers
	for i := 0; i < len(rooms)-1; i++ {
		r1 := rooms[i]
		r2 := rooms[i+1]

		// Find center of each room
		cx1 := r1.x + r1.w/2
		cy1 := r1.y + r1.h/2
		cx2 := r2.x + r2.w/2
		cy2 := r2.y + r2.h/2

		// Draw L-shaped corridor
		// First go horizontal, then vertical (or vice versa randomly)
		if rng.Intn(2) == 0 {
			// Horizontal first
			for x := min(cx1, cx2); x <= max(cx1, cx2); x++ {
				carve(x, cy1)
				if x > min(cx1, cx2) {
					carvePassage(x-1, cy1, x, cy1)
				}
				visited[cy1][x] = true
			}
			for y := min(cy1, cy2); y <= max(cy1, cy2); y++ {
				carve(cx2, y)
				if y > min(cy1, cy2) {
					carvePassage(cx2, y-1, cx2, y)
				}
				visited[y][cx2] = true
			}
		} else {
			// Vertical first
			for y := min(cy1, cy2); y <= max(cy1, cy2); y++ {
				carve(cx1, y)
				if y > min(cy1, cy2) {
					carvePassage(cx1, y-1, cx1, y)
				}
				visited[y][cx1] = true
			}
			for x := min(cx1, cx2); x <= max(cx1, cx2); x++ {
				carve(x, cy2)
				if x > min(cx1, cx2) {
					carvePassage(x-1, cy2, x, cy2)
				}
				visited[cy2][x] = true
			}
		}
	}

	// Connect any remaining unvisited cells
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for y := 0; y < logicHeight; y++ {
		for x := 0; x < logicWidth; x++ {
			if !visited[y][x] {
				// Find nearest visited cell and connect
				carve(x, y)
				visited[y][x] = true
				// Try to connect to adjacent visited cell
				for _, dir := range dirs {
					nx, ny := x+dir[0], y+dir[1]
					if nx >= 0 && ny >= 0 && nx < logicWidth && ny < logicHeight && visited[ny][nx] {
						carvePassage(x, y, nx, ny)
						break
					}
				}
			}
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

// GenerateSpiral creates a spiral-like maze pattern
func GenerateSpiral(logicWidth, logicHeight int, rng *rand.Rand) Grid {
	if logicWidth < 3 {
		logicWidth = 3
	}
	if logicHeight < 3 {
		logicHeight = 3
	}

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

	// Helper to carve a corridor cell
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
			minY := sy1
			if sy2 < sy1 {
				minY = sy2
			}
			for y := minY; y < minY+CellSize+CorridorWidth; y++ {
				for dx := 0; dx < CorridorWidth; dx++ {
					if y < screenHeight && sx1+dx < screenWidth {
						cells[y][sx1+dx] = Floor
					}
				}
			}
		} else {
			minX := sx1
			if sx2 < sx1 {
				minX = sx2
			}
			for x := minX; x < minX+CellSize+CorridorWidth; x++ {
				for dy := 0; dy < CorridorWidth; dy++ {
					if sy1+dy < screenHeight && x < screenWidth {
						cells[sy1+dy][x] = Floor
					}
				}
			}
		}
	}

	// Track visited cells
	visited := make([][]bool, logicHeight)
	for y := 0; y < logicHeight; y++ {
		visited[y] = make([]bool, logicWidth)
	}

	// Spiral from outside to inside
	// Direction order: right, down, left, up
	dirs := [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}
	dirIdx := 0

	x, y := 0, 0
	carve(x, y)
	visited[y][x] = true

	for {
		// Try current direction
		dx, dy := dirs[dirIdx][0], dirs[dirIdx][1]
		nx, ny := x+dx, y+dy

		// Check if we can continue in current direction
		if nx >= 0 && ny >= 0 && nx < logicWidth && ny < logicHeight && !visited[ny][nx] {
			carve(nx, ny)
			carvePassage(x, y, nx, ny)
			visited[ny][nx] = true
			x, y = nx, ny
		} else {
			// Try turning (change direction)
			turned := false
			for i := 0; i < 4; i++ {
				dirIdx = (dirIdx + 1) % 4
				dx, dy = dirs[dirIdx][0], dirs[dirIdx][1]
				nx, ny = x+dx, y+dy
				if nx >= 0 && ny >= 0 && nx < logicWidth && ny < logicHeight && !visited[ny][nx] {
					carve(nx, ny)
					carvePassage(x, y, nx, ny)
					visited[ny][nx] = true
					x, y = nx, ny
					turned = true
					break
				}
			}
			if !turned {
				break // No more moves possible
			}
		}
	}

	// Add some random connections to make it more interesting (30% of cells)
	extraConnections := (logicWidth * logicHeight) / 3
	for i := 0; i < extraConnections; i++ {
		rx := rng.Intn(logicWidth - 1)
		ry := rng.Intn(logicHeight - 1)
		if rng.Intn(2) == 0 && rx+1 < logicWidth {
			carvePassage(rx, ry, rx+1, ry)
		} else if ry+1 < logicHeight {
			carvePassage(rx, ry, rx, ry+1)
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

// GenerateGrid creates a regular grid pattern with random walls removed
func GenerateGrid(logicWidth, logicHeight int, rng *rand.Rand) Grid {
	if logicWidth < 3 {
		logicWidth = 3
	}
	if logicHeight < 3 {
		logicHeight = 3
	}

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

	// Helper to carve a corridor cell
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
			minY := sy1
			if sy2 < sy1 {
				minY = sy2
			}
			for y := minY; y < minY+CellSize+CorridorWidth; y++ {
				for dx := 0; dx < CorridorWidth; dx++ {
					if y < screenHeight && sx1+dx < screenWidth {
						cells[y][sx1+dx] = Floor
					}
				}
			}
		} else {
			minX := sx1
			if sx2 < sx1 {
				minX = sx2
			}
			for x := minX; x < minX+CellSize+CorridorWidth; x++ {
				for dy := 0; dy < CorridorWidth; dy++ {
					if sy1+dy < screenHeight && x < screenWidth {
						cells[sy1+dy][x] = Floor
					}
				}
			}
		}
	}

	// Carve all cells first
	for ly := 0; ly < logicHeight; ly++ {
		for lx := 0; lx < logicWidth; lx++ {
			carve(lx, ly)
		}
	}

	// Create a grid pattern: passages every 2 cells (checkered pattern)
	// Horizontal passages on even rows
	for ly := 0; ly < logicHeight; ly++ {
		for lx := 0; lx < logicWidth-1; lx++ {
			// Always connect on even rows, 50% on odd rows
			if ly%2 == 0 || rng.Float64() < 0.5 {
				carvePassage(lx, ly, lx+1, ly)
			}
		}
	}

	// Vertical passages on even columns
	for ly := 0; ly < logicHeight-1; ly++ {
		for lx := 0; lx < logicWidth; lx++ {
			// Always connect on even columns, 50% on odd columns
			if lx%2 == 0 || rng.Float64() < 0.5 {
				carvePassage(lx, ly, lx, ly+1)
			}
		}
	}

	// Ensure full connectivity using union-find or DFS
	// Simple approach: DFS from (0,0) and connect any unreachable cells
	visited := make([][]bool, logicHeight)
	for y := 0; y < logicHeight; y++ {
		visited[y] = make([]bool, logicWidth)
	}

	var dfs func(x, y int)
	dfs = func(x, y int) {
		if x < 0 || y < 0 || x >= logicWidth || y >= logicHeight || visited[y][x] {
			return
		}
		visited[y][x] = true
		// Check if passage exists to neighbors
		grid := &Grid{Width: screenWidth, Height: screenHeight, LogicWidth: logicWidth, LogicHeight: logicHeight, Cells: cells}
		dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && ny >= 0 && nx < logicWidth && ny < logicHeight && !visited[ny][nx] {
				if grid.CanMove(x, y, nx, ny) {
					dfs(nx, ny)
				}
			}
		}
	}

	dfs(0, 0)

	// Connect any unvisited cells
	for ly := 0; ly < logicHeight; ly++ {
		for lx := 0; lx < logicWidth; lx++ {
			if !visited[ly][lx] {
				// Connect to a visited neighbor
				dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
				for _, d := range dirs {
					nx, ny := lx+d[0], ly+d[1]
					if nx >= 0 && ny >= 0 && nx < logicWidth && ny < logicHeight && visited[ny][nx] {
						carvePassage(lx, ly, nx, ny)
						dfs(lx, ly) // Re-run DFS from newly connected cell
						break
					}
				}
			}
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

// GenerateWithStyle creates a maze using the specified style
func GenerateWithStyle(logicWidth, logicHeight int, rng *rand.Rand, style MazeStyle) Grid {
	switch style {
	case StyleCorridors:
		return GenerateWithStraightBias(logicWidth, logicHeight, rng, 0.85)
	case StyleRooms:
		return GenerateRooms(logicWidth, logicHeight, rng)
	case StyleSpiral:
		return GenerateSpiral(logicWidth, logicHeight, rng)
	case StyleGrid:
		return GenerateGrid(logicWidth, logicHeight, rng)
	default: // StyleClassic
		return Generate(logicWidth, logicHeight, rng)
	}
}

// HasPath checks if there's a path between two logical positions using BFS
func (g *Grid) HasPath(startX, startY, endX, endY int) bool {
	if !g.IsFloor(startX, startY) || !g.IsFloor(endX, endY) {
		return false
	}

	visited := make([][]bool, g.LogicHeight)
	for y := 0; y < g.LogicHeight; y++ {
		visited[y] = make([]bool, g.LogicWidth)
	}

	type pos struct{ x, y int }
	queue := []pos{{startX, startY}}
	visited[startY][startX] = true

	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.x == endX && current.y == endY {
			return true
		}

		for _, d := range dirs {
			nx, ny := current.x+d[0], current.y+d[1]
			if nx >= 0 && ny >= 0 && nx < g.LogicWidth && ny < g.LogicHeight && !visited[ny][nx] {
				if g.CanMove(current.x, current.y, nx, ny) {
					visited[ny][nx] = true
					queue = append(queue, pos{nx, ny})
				}
			}
		}
	}

	return false
}

// FindPath returns a path between two logical positions using BFS, or nil if no path exists
func (g *Grid) FindPath(startX, startY, endX, endY int) [][2]int {
	if !g.IsFloor(startX, startY) || !g.IsFloor(endX, endY) {
		return nil
	}

	visited := make([][]bool, g.LogicHeight)
	parent := make([][][2]int, g.LogicHeight)
	for y := 0; y < g.LogicHeight; y++ {
		visited[y] = make([]bool, g.LogicWidth)
		parent[y] = make([][2]int, g.LogicWidth)
		for x := 0; x < g.LogicWidth; x++ {
			parent[y][x] = [2]int{-1, -1}
		}
	}

	type pos struct{ x, y int }
	queue := []pos{{startX, startY}}
	visited[startY][startX] = true

	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.x == endX && current.y == endY {
			// Reconstruct path
			path := [][2]int{}
			cx, cy := endX, endY
			for cx != -1 && cy != -1 {
				path = append([][2]int{{cx, cy}}, path...)
				p := parent[cy][cx]
				cx, cy = p[0], p[1]
			}
			return path
		}

		for _, d := range dirs {
			nx, ny := current.x+d[0], current.y+d[1]
			if nx >= 0 && ny >= 0 && nx < g.LogicWidth && ny < g.LogicHeight && !visited[ny][nx] {
				if g.CanMove(current.x, current.y, nx, ny) {
					visited[ny][nx] = true
					parent[ny][nx] = [2]int{current.x, current.y}
					queue = append(queue, pos{nx, ny})
				}
			}
		}
	}

	return nil
}

// EnsureConnectivity carves a path between two points if no path exists
// This guarantees the maze is traversable between entry and exit
func (g *Grid) EnsureConnectivity(startX, startY, endX, endY int) {
	if g.HasPath(startX, startY, endX, endY) {
		return
	}

	// No path exists - carve one using A*-like approach (prefer straight lines)
	// Use BFS on all cells (ignoring walls) to find shortest logical path, then carve it

	visited := make([][]bool, g.LogicHeight)
	parent := make([][][2]int, g.LogicHeight)
	for y := 0; y < g.LogicHeight; y++ {
		visited[y] = make([]bool, g.LogicWidth)
		parent[y] = make([][2]int, g.LogicWidth)
		for x := 0; x < g.LogicWidth; x++ {
			parent[y][x] = [2]int{-1, -1}
		}
	}

	type pos struct{ x, y int }
	queue := []pos{{startX, startY}}
	visited[startY][startX] = true

	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.x == endX && current.y == endY {
			break
		}

		for _, d := range dirs {
			nx, ny := current.x+d[0], current.y+d[1]
			if nx >= 0 && ny >= 0 && nx < g.LogicWidth && ny < g.LogicHeight && !visited[ny][nx] {
				visited[ny][nx] = true
				parent[ny][nx] = [2]int{current.x, current.y}
				queue = append(queue, pos{nx, ny})
			}
		}
	}

	// Reconstruct and carve path
	cx, cy := endX, endY
	for cx != -1 && cy != -1 && (cx != startX || cy != startY) {
		// Carve this cell
		g.carveCell(cx, cy)

		p := parent[cy][cx]
		px, py := p[0], p[1]
		if px != -1 && py != -1 {
			// Carve passage between current and parent
			g.carvePassageBetween(px, py, cx, cy)
		}
		cx, cy = px, py
	}
	// Carve start cell too
	g.carveCell(startX, startY)
}

// carveCell carves a single logical cell (makes it floor)
func (g *Grid) carveCell(lx, ly int) {
	sx, sy := LogicToScreen(lx, ly)
	for dy := 0; dy < CorridorWidth; dy++ {
		for dx := 0; dx < CorridorWidth; dx++ {
			if sy+dy < g.Height && sx+dx < g.Width {
				g.Cells[sy+dy][sx+dx] = Floor
			}
		}
	}
}

// carvePassageBetween carves a passage between two adjacent logical cells
func (g *Grid) carvePassageBetween(lx1, ly1, lx2, ly2 int) {
	sx1, sy1 := LogicToScreen(lx1, ly1)
	sx2, sy2 := LogicToScreen(lx2, ly2)

	if lx1 == lx2 {
		// Vertical passage
		minY := sy1
		if sy2 < sy1 {
			minY = sy2
		}
		for y := minY; y < minY+CellSize+CorridorWidth; y++ {
			for dx := 0; dx < CorridorWidth; dx++ {
				if y < g.Height && sx1+dx < g.Width {
					g.Cells[y][sx1+dx] = Floor
				}
			}
		}
	} else {
		// Horizontal passage
		minX := sx1
		if sx2 < sx1 {
			minX = sx2
		}
		for x := minX; x < minX+CellSize+CorridorWidth; x++ {
			for dy := 0; dy < CorridorWidth; dy++ {
				if sy1+dy < g.Height && x < g.Width {
					g.Cells[sy1+dy][x] = Floor
				}
			}
		}
	}
}
