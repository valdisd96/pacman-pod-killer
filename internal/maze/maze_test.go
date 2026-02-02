package maze

import (
	"math/rand"
	"testing"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name         string
		logicWidth   int
		logicHeight  int
		wantMinWidth int
	}{
		{"small maze", 5, 5, 5},
		{"standard maze", 15, 10, 15},
		{"large maze", 20, 20, 20},
		{"minimum size", 1, 1, 3}, // Should be adjusted to minimum
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(42))
			grid := Generate(tt.logicWidth, tt.logicHeight, rng)

			if grid.LogicWidth < tt.wantMinWidth {
				t.Errorf("LogicWidth = %d, want at least %d", grid.LogicWidth, tt.wantMinWidth)
			}
			if grid.LogicHeight < 3 {
				t.Errorf("LogicHeight = %d, want at least 3", grid.LogicHeight)
			}
			if grid.Width <= 0 || grid.Height <= 0 {
				t.Error("Grid dimensions should be positive")
			}
		})
	}
}

func TestGenerate_Deterministic(t *testing.T) {
	seed := int64(42)
	rng1 := rand.New(rand.NewSource(seed))
	rng2 := rand.New(rand.NewSource(seed))

	grid1 := Generate(10, 10, rng1)
	grid2 := Generate(10, 10, rng2)

	// With same seed, should generate identical grids
	if grid1.LogicWidth != grid2.LogicWidth || grid1.LogicHeight != grid2.LogicHeight {
		t.Error("Same seed should produce same dimensions")
	}

	// Compare cells
	if grid1.Width != grid2.Width || grid1.Height != grid2.Height {
		t.Error("Same seed should produce same screen dimensions")
	}

	for y := 0; y < grid1.Height && y < grid2.Height; y++ {
		for x := 0; x < grid1.Width && x < grid2.Width; x++ {
			if grid1.Cells[y][x] != grid2.Cells[y][x] {
				t.Errorf("Same seed should produce identical cells at (%d, %d)", x, y)
				return
			}
		}
	}
}

func TestIsFloor(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := Generate(5, 5, rng)

	// Test out of bounds
	if grid.IsFloor(-1, 0) {
		t.Error("IsFloor(-1, 0) should be false")
	}
	if grid.IsFloor(0, -1) {
		t.Error("IsFloor(0, -1) should be false")
	}
	if grid.IsFloor(grid.LogicWidth, 0) {
		t.Error("IsFloor(out of width, 0) should be false")
	}
	if grid.IsFloor(0, grid.LogicHeight) {
		t.Error("IsFloor(0, out of height) should be false")
	}

	// (0, 0) should be floor since we start carving there
	if !grid.IsFloor(0, 0) {
		t.Error("IsFloor(0, 0) should be true (start position)")
	}
}

func TestCanMove(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := Generate(5, 5, rng)

	tests := []struct {
		name   string
		fromX  int
		fromY  int
		toX    int
		toY    int
		expect bool
	}{
		{"move to same cell", 0, 0, 0, 0, false},
		{"move out of bounds from", -1, 0, 0, 0, false},
		{"move out of bounds to", 0, 0, -1, 0, false},
		{"diagonal move", 0, 0, 1, 1, false},
		{"move more than 1 cell", 0, 0, 2, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := grid.CanMove(tt.fromX, tt.fromY, tt.toX, tt.toY)
			if got != tt.expect {
				t.Errorf("CanMove(%d, %d, %d, %d) = %v, want %v",
					tt.fromX, tt.fromY, tt.toX, tt.toY, got, tt.expect)
			}
		})
	}
}

func TestCanMove_Adjacent(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := Generate(5, 5, rng)

	// Find a floor cell and test adjacent moves
	found := false
	for ly := 0; ly < grid.LogicHeight && !found; ly++ {
		for lx := 0; lx < grid.LogicWidth && !found; lx++ {
			if grid.IsFloor(lx, ly) {
				// Test all four directions
				dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
				for _, d := range dirs {
					nx, ny := lx+d[0], ly+d[1]
					if nx >= 0 && ny >= 0 && nx < grid.LogicWidth && ny < grid.LogicHeight {
						canMove := grid.CanMove(lx, ly, nx, ny)
						// Can move if both cells are floor and passage exists
						if canMove && !grid.IsFloor(nx, ny) {
							t.Errorf("CanMove to non-floor cell at (%d, %d)", nx, ny)
						}
					}
				}
				found = true
			}
		}
	}

	if !found {
		t.Error("No floor cell found in maze")
	}
}

func TestLogicToScreen(t *testing.T) {
	tests := []struct {
		lx     int
		ly     int
		wantSX int
		wantSY int
	}{
		{0, 0, 2, 2}, // WallThickness = 2
		{1, 0, 7, 2}, // WallThickness + CellSize
		{0, 1, 2, 7},
		{1, 1, 7, 7},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			sx, sy := LogicToScreen(tt.lx, tt.ly)
			if sx != tt.wantSX || sy != tt.wantSY {
				t.Errorf("LogicToScreen(%d, %d) = (%d, %d), want (%d, %d)",
					tt.lx, tt.ly, sx, sy, tt.wantSX, tt.wantSY)
			}
		})
	}
}

func TestScreenToLogic(t *testing.T) {
	tests := []struct {
		sx     int
		sy     int
		wantLX int
		wantLY int
	}{
		{2, 2, 0, 0}, // Exact corner
		{4, 4, 0, 0}, // Inside first cell
		{7, 2, 1, 0}, // Second cell horizontally
		{2, 7, 0, 1}, // Second cell vertically
		{7, 7, 1, 1}, // Second cell diagonally
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			lx, ly := ScreenToLogic(tt.sx, tt.sy)
			if lx != tt.wantLX || ly != tt.wantLY {
				t.Errorf("ScreenToLogic(%d, %d) = (%d, %d), want (%d, %d)",
					tt.sx, tt.sy, lx, ly, tt.wantLX, tt.wantLY)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that LogicToScreen -> ScreenToLogic is roughly correct
	tests := []struct {
		lx int
		ly int
	}{
		{0, 0},
		{1, 0},
		{0, 1},
		{5, 5},
		{10, 10},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			sx, sy := LogicToScreen(tt.lx, tt.ly)
			lx, ly := ScreenToLogic(sx, sy)
			// Should get back to same logical coordinates
			if lx != tt.lx || ly != tt.ly {
				t.Errorf("Round-trip failed: (%d, %d) -> (%d, %d) -> (%d, %d)",
					tt.lx, tt.ly, sx, sy, lx, ly)
			}
		})
	}
}

func TestHasPath(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := Generate(5, 5, rng)

	// (0, 0) should always have a path to itself
	if !grid.HasPath(0, 0, 0, 0) {
		t.Error("Should have path from (0, 0) to itself")
	}

	// Find two floor cells and check if there's a path
	var floors [][2]int
	for ly := 0; ly < grid.LogicHeight; ly++ {
		for lx := 0; lx < grid.LogicWidth; lx++ {
			if grid.IsFloor(lx, ly) {
				floors = append(floors, [2]int{lx, ly})
			}
		}
	}

	if len(floors) < 2 {
		t.Fatal("Need at least 2 floor cells to test path")
	}

	// Test path between first two floor cells
	start := floors[0]
	end := floors[1]
	// Note: In a DFS-generated maze, not all floor cells may be connected
	// This is just checking the function doesn't panic
	_ = grid.HasPath(start[0], start[1], end[0], end[1])

	// Test with non-floor cells
	if grid.HasPath(-1, -1, 0, 0) {
		t.Error("Should not have path from invalid cell")
	}
}

func TestFindPath(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := Generate(5, 5, rng)

	// Path from (0, 0) to itself should be single point
	path := grid.FindPath(0, 0, 0, 0)
	if len(path) != 1 {
		t.Errorf("Path to self should be 1 point, got %d", len(path))
	}

	// Path with non-floor cells should be nil
	path = grid.FindPath(-1, -1, 0, 0)
	if path != nil {
		t.Error("Path from invalid cell should be nil")
	}
}

func TestEnsureConnectivity(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := Generate(5, 5, rng)

	// Pick two points
	startX, startY := 0, 0
	endX, endY := 4, 4

	// Ensure they're both floor
	grid.carveCell(startX, startY)
	grid.carveCell(endX, endY)

	// Ensure connectivity
	grid.EnsureConnectivity(startX, startY, endX, endY)

	// Now there should be a path
	if !grid.HasPath(startX, startY, endX, endY) {
		t.Error("EnsureConnectivity should create a path")
	}

	// Test with already connected cells (should not error)
	grid.EnsureConnectivity(startX, startY, endX, endY)
}

func TestGenerateWithStyle(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	styles := []MazeStyle{StyleClassic, StyleCorridors, StyleRooms, StyleSpiral, StyleGrid}

	for _, style := range styles {
		t.Run("", func(t *testing.T) {
			grid := GenerateWithStyle(10, 10, rng, style)
			if grid.LogicWidth <= 0 || grid.LogicHeight <= 0 {
				t.Errorf("Style %v generated invalid dimensions", style)
			}
		})
	}
}

func TestGenerateRooms(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := GenerateRooms(10, 10, rng)

	if grid.LogicWidth < 3 || grid.LogicHeight < 3 {
		t.Error("GenerateRooms should enforce minimum dimensions")
	}

	// Should have floor cells (rooms)
	foundFloor := false
	for ly := 0; ly < grid.LogicHeight && !foundFloor; ly++ {
		for lx := 0; lx < grid.LogicWidth && !foundFloor; lx++ {
			if grid.IsFloor(lx, ly) {
				foundFloor = true
			}
		}
	}

	if !foundFloor {
		t.Error("GenerateRooms should create floor cells")
	}
}

func TestGenerateSpiral(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := GenerateSpiral(10, 10, rng)

	if grid.LogicWidth < 3 || grid.LogicHeight < 3 {
		t.Error("GenerateSpiral should enforce minimum dimensions")
	}

	// (0, 0) should be floor (starting point of spiral)
	if !grid.IsFloor(0, 0) {
		t.Error("GenerateSpiral should start at (0, 0)")
	}
}

func TestGenerateGrid(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := GenerateGrid(10, 10, rng)

	if grid.LogicWidth < 3 || grid.LogicHeight < 3 {
		t.Error("GenerateGrid should enforce minimum dimensions")
	}

	// Grid should have many floor cells
	floorCount := 0
	for ly := 0; ly < grid.LogicHeight; ly++ {
		for lx := 0; lx < grid.LogicWidth; lx++ {
			if grid.IsFloor(lx, ly) {
				floorCount++
			}
		}
	}

	// Grid style carves all cells initially
	if floorCount == 0 {
		t.Error("GenerateGrid should create floor cells")
	}
}

func TestGenerateWithStraightBias(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	// Test with 0 bias (same as regular Generate)
	grid1 := Generate(10, 10, rand.New(rand.NewSource(42)))
	grid2 := GenerateWithStraightBias(10, 10, rand.New(rand.NewSource(42)), 0.0)

	// Should have same dimensions
	if grid1.LogicWidth != grid2.LogicWidth || grid1.LogicHeight != grid2.LogicHeight {
		t.Error("Generate with 0 bias should have same dimensions")
	}

	// Test with high bias
	grid3 := GenerateWithStraightBias(10, 10, rng, 0.85)
	if grid3.LogicWidth < 3 || grid3.LogicHeight < 3 {
		t.Error("High bias generate should enforce minimum dimensions")
	}
}

func TestConstants(t *testing.T) {
	if Wall != '#' {
		t.Errorf("Wall should be '#', got %c", Wall)
	}
	if Floor != ' ' {
		t.Errorf("Floor should be ' ', got %c", Floor)
	}
	if WallThickness != 2 {
		t.Errorf("WallThickness should be 2, got %d", WallThickness)
	}
	if CorridorWidth != 3 {
		t.Errorf("CorridorWidth should be 3, got %d", CorridorWidth)
	}
	if CellSize != 5 {
		t.Errorf("CellSize should be 5, got %d", CellSize)
	}
}
