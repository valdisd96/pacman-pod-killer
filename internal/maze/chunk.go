package maze

import (
	"math/rand"
)

// MazeStyle represents different maze generation patterns
type MazeStyle int

const (
	StyleClassic   MazeStyle = iota // Standard random DFS maze
	StyleCorridors                  // Long straight corridors (high straightBias)
	StyleRooms                      // Open rooms connected by passages
	StyleSpiral                     // Spiral-like pattern from outside to center
	StyleGrid                       // Regular grid with random walls removed
)

// Location represents a named level/area in the game
type Location struct {
	Name     string    // Location name (e.g., "location1")
	Index    int       // Position in the sequence (0, 1, 2, ...)
	IsFinal  bool      // True if this is the winning location
	ExitEdge Edge      // Which edge has the exit to next location
	Style    MazeStyle // Maze generation style for this location
}

// Edge represents a maze boundary
type Edge int

const (
	EdgeTop Edge = iota
	EdgeBottom
	EdgeLeft
	EdgeRight
)

// Portal represents an exit/entry point between locations
type Portal struct {
	X         int    // Logical X position
	Y         int    // Logical Y position
	Edge      Edge   // Which edge the portal is on
	IsExit    bool   // True if this is an exit to next location
	IsEntry   bool   // True if this is an entry from previous location
	IsWinExit bool   // True if this is the final exit (triggers game win)
	TargetLoc string // Target location name
}

// ChunkManager manages maze chunks for each location
type ChunkManager struct {
	chunks       map[string]*Grid    // Keyed by location name
	portals      map[string][]Portal // Portals per location
	locations    []Location          // Ordered list of locations
	logicWidth   int
	logicHeight  int
	straightBias float64 // Probability to continue in same direction (0.0-1.0)
}

// randomStyle returns a random maze style
func randomStyle(rng *rand.Rand) MazeStyle {
	styles := []MazeStyle{StyleClassic, StyleCorridors, StyleRooms, StyleSpiral, StyleGrid}
	return styles[rng.Intn(len(styles))]
}

// NewChunkManager creates a chunk manager for the given locations
// locationNames is a comma-separated or slice of location patterns (e.g., ["location1", "location2", "location3"])
func NewChunkManager(locationNames []string, logicWidth, logicHeight int, rng *rand.Rand) *ChunkManager {
	locations := make([]Location, len(locationNames))
	for i, name := range locationNames {
		locations[i] = Location{
			Name:     name,
			Index:    i,
			IsFinal:  i == len(locationNames)-1,
			ExitEdge: EdgeRight, // Default: exit on right edge to next location
			Style:    randomStyle(rng),
		}
	}

	return &ChunkManager{
		chunks:       make(map[string]*Grid),
		portals:      make(map[string][]Portal),
		locations:    locations,
		logicWidth:   logicWidth,
		logicHeight:  logicHeight,
		straightBias: 0.7, // 70% chance to continue in same direction (used for StyleCorridors)
	}
}

// GetLocation returns the location by name
func (cm *ChunkManager) GetLocation(name string) *Location {
	for i := range cm.locations {
		if cm.locations[i].Name == name {
			return &cm.locations[i]
		}
	}
	return nil
}

// GetLocationByIndex returns the location at the given index
func (cm *ChunkManager) GetLocationByIndex(index int) *Location {
	if index < 0 || index >= len(cm.locations) {
		return nil
	}
	return &cm.locations[index]
}

// NextLocation returns the next location after the given one, or nil if at the end
func (cm *ChunkManager) NextLocation(current string) *Location {
	for i, loc := range cm.locations {
		if loc.Name == current && i+1 < len(cm.locations) {
			return &cm.locations[i+1]
		}
	}
	return nil
}

// PrevLocation returns the previous location before the given one, or nil if at the start
func (cm *ChunkManager) PrevLocation(current string) *Location {
	for i, loc := range cm.locations {
		if loc.Name == current && i > 0 {
			return &cm.locations[i-1]
		}
	}
	return nil
}

// GetOrGenerate returns existing maze for a location or generates a new one
func (cm *ChunkManager) GetOrGenerate(locationName string, rng *rand.Rand) *Grid {
	if grid, exists := cm.chunks[locationName]; exists {
		return grid
	}

	// Find the location to determine edge connections
	loc := cm.GetLocation(locationName)
	if loc == nil {
		// Unknown location, generate without edges
		grid := Generate(cm.logicWidth, cm.logicHeight, rng)
		return &grid
	}

	// Generate the base maze using the location's style
	grid := GenerateWithStyle(cm.logicWidth, cm.logicHeight, rng, loc.Style)

	// Create portals for this location
	var portals []Portal

	// Exit portal on right edge
	if !loc.IsFinal {
		// Regular exit to next location
		nextLoc := cm.NextLocation(locationName)
		targetName := ""
		if nextLoc != nil {
			targetName = nextLoc.Name
		}
		exitY := cm.carveEdgePassage(&grid, EdgeRight, rng)
		portals = append(portals, Portal{
			X:         grid.LogicWidth - 1,
			Y:         exitY,
			Edge:      EdgeRight,
			IsExit:    true,
			IsEntry:   false,
			IsWinExit: false,
			TargetLoc: targetName,
		})
	} else {
		// Final location: WIN exit portal
		exitY := cm.carveEdgePassage(&grid, EdgeRight, rng)
		portals = append(portals, Portal{
			X:         grid.LogicWidth - 1,
			Y:         exitY,
			Edge:      EdgeRight,
			IsExit:    true,
			IsEntry:   false,
			IsWinExit: true,
			TargetLoc: "WIN",
		})
	}

	// Entry from previous location (left edge, except for first location)
	var entryX, entryY int
	if loc.Index > 0 {
		prevLoc := cm.PrevLocation(locationName)
		targetName := ""
		if prevLoc != nil {
			targetName = prevLoc.Name
		}
		entryY = cm.carveEdgePassage(&grid, EdgeLeft, rng)
		entryX = 0
		portals = append(portals, Portal{
			X:         entryX,
			Y:         entryY,
			Edge:      EdgeLeft,
			IsExit:    false,
			IsEntry:   true,
			TargetLoc: targetName,
		})
	} else {
		// First location: entry point is (0, 0) or any accessible cell
		entryX = 0
		entryY = 0
	}

	// Find exit portal position
	var exitX, exitY int
	for _, p := range portals {
		if p.IsExit {
			exitX = p.X
			exitY = p.Y
			break
		}
	}

	// Ensure there's a path from entry to exit
	// This guarantees the maze is always solvable
	grid.EnsureConnectivity(entryX, entryY, exitX, exitY)

	cm.chunks[locationName] = &grid
	cm.portals[locationName] = portals
	return &grid
}

// FirstLocation returns the first location name
func (cm *ChunkManager) FirstLocation() string {
	if len(cm.locations) == 0 {
		return ""
	}
	return cm.locations[0].Name
}

// LocationCount returns the number of locations
func (cm *ChunkManager) LocationCount() int {
	return len(cm.locations)
}

// Locations returns all location names in order
func (cm *ChunkManager) Locations() []string {
	names := make([]string, len(cm.locations))
	for i, loc := range cm.locations {
		names[i] = loc.Name
	}
	return names
}

// GetPortals returns the portals for a location
func (cm *ChunkManager) GetPortals(locationName string) []Portal {
	return cm.portals[locationName]
}

// GetExitPortal returns the exit portal for a location (portal to next location)
func (cm *ChunkManager) GetExitPortal(locationName string) *Portal {
	for i := range cm.portals[locationName] {
		if cm.portals[locationName][i].IsExit {
			return &cm.portals[locationName][i]
		}
	}
	return nil
}

// GetEntryPortal returns the entry portal for a location (portal from previous location)
func (cm *ChunkManager) GetEntryPortal(locationName string) *Portal {
	for i := range cm.portals[locationName] {
		if cm.portals[locationName][i].IsEntry {
			return &cm.portals[locationName][i]
		}
	}
	return nil
}

// GetWinPortal returns the WIN portal for a location (only on final location)
func (cm *ChunkManager) GetWinPortal(locationName string) *Portal {
	for i := range cm.portals[locationName] {
		if cm.portals[locationName][i].IsWinExit {
			return &cm.portals[locationName][i]
		}
	}
	return nil
}

// carveEdgePassage creates a passage at the specified edge of the maze
// Returns the position (Y for left/right edges, X for top/bottom edges)
func (cm *ChunkManager) carveEdgePassage(grid *Grid, edge Edge, rng *rand.Rand) int {
	var lx, ly int

	switch edge {
	case EdgeRight:
		// Passage on right edge - pick a random row
		lx = grid.LogicWidth - 1
		ly = rng.Intn(grid.LogicHeight)
	case EdgeLeft:
		// Passage on left edge - pick a random row
		lx = 0
		ly = rng.Intn(grid.LogicHeight)
	case EdgeTop:
		// Passage on top edge - pick a random column
		lx = rng.Intn(grid.LogicWidth)
		ly = 0
	case EdgeBottom:
		// Passage on bottom edge - pick a random column
		lx = rng.Intn(grid.LogicWidth)
		ly = grid.LogicHeight - 1
	}

	// The cell is already carved by maze generation
	// Just return the position for portal tracking
	_ = lx

	// Return Y for horizontal edges, X for vertical edges
	if edge == EdgeRight || edge == EdgeLeft {
		return ly
	}
	return lx
}

// HasExitAt checks if there's a floor tile at the edge for teleportation
func (cm *ChunkManager) HasExitAt(grid *Grid, edge Edge, pos int) bool {
	var lx, ly int

	switch edge {
	case EdgeRight:
		lx = grid.LogicWidth - 1
		ly = pos
	case EdgeLeft:
		lx = 0
		ly = pos
	case EdgeTop:
		lx = pos
		ly = 0
	case EdgeBottom:
		lx = pos
		ly = grid.LogicHeight - 1
	}

	return grid.IsFloor(lx, ly)
}

// FindExitPosition finds a valid exit position on the given edge
func (cm *ChunkManager) FindExitPosition(grid *Grid, edge Edge, preferredPos int) int {
	var maxPos int
	switch edge {
	case EdgeRight, EdgeLeft:
		maxPos = grid.LogicHeight
	case EdgeTop, EdgeBottom:
		maxPos = grid.LogicWidth
	}

	// First try the preferred position
	if preferredPos >= 0 && preferredPos < maxPos && cm.HasExitAt(grid, edge, preferredPos) {
		return preferredPos
	}

	// Search outward from preferred position
	for delta := 1; delta < maxPos; delta++ {
		pos1 := preferredPos + delta
		pos2 := preferredPos - delta
		if pos1 < maxPos && cm.HasExitAt(grid, edge, pos1) {
			return pos1
		}
		if pos2 >= 0 && cm.HasExitAt(grid, edge, pos2) {
			return pos2
		}
	}

	// Fallback to 0
	return 0
}
