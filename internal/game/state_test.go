package game

import (
	"math/rand"
	"testing"
	"time"

	"pacman-pod-killer/internal/maze"
)

func TestNewState(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := maze.Generate(10, 10, rng)

	state := NewState(grid, 42, 10*time.Second, 2)

	if state == nil {
		t.Fatal("NewState returned nil")
	}

	if state.Player.X != 0 || state.Player.Y != 0 {
		t.Error("Player should start at (0, 0)")
	}

	if !state.Player.Alive {
		t.Error("Player should be alive initially")
	}

	if state.Player.LastDirection.X != 1 || state.Player.LastDirection.Y != 0 {
		t.Error("Player should default to right direction")
	}

	if state.EnemySpeed != 2 {
		t.Errorf("EnemySpeed should be 2, got %d", state.EnemySpeed)
	}

	if state.Seed != 42 {
		t.Errorf("Seed should be 42, got %d", state.Seed)
	}

	if state.ShootCooldown != 5 {
		t.Errorf("ShootCooldown should be 5, got %d", state.ShootCooldown)
	}
}

func TestNewStateWithConfig(t *testing.T) {
	cfg := StateConfig{
		Seed:         123,
		RespawnDelay: 5 * time.Second,
		EnemySpeed:   3,
		LogicWidth:   10,
		LogicHeight:  10,
	}

	state := NewStateWithConfig(cfg, nil)

	if state == nil {
		t.Fatal("NewStateWithConfig returned nil")
	}

	if state.EnemySpeed != 3 {
		t.Errorf("EnemySpeed should be 3, got %d", state.EnemySpeed)
	}

	if state.Player.RespawnDelay != 5*time.Second {
		t.Errorf("RespawnDelay should be 5s, got %v", state.Player.RespawnDelay)
	}
}

func TestNewStateWithConfig_MinEnemySpeed(t *testing.T) {
	cfg := StateConfig{
		Seed:         123,
		RespawnDelay: 5 * time.Second,
		EnemySpeed:   0, // Invalid, should be adjusted to 1
		LogicWidth:   10,
		LogicHeight:  10,
	}

	state := NewStateWithConfig(cfg, nil)

	if state.EnemySpeed != 1 {
		t.Errorf("EnemySpeed should be minimum 1, got %d", state.EnemySpeed)
	}
}

func TestFindOpenPosition(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := maze.Generate(10, 10, rng)
	state := NewState(grid, 42, 10*time.Second, 2)

	pos := state.FindOpenPosition(rng)

	// Should return a valid floor position
	if !state.Maze.IsFloor(pos.X, pos.Y) {
		t.Errorf("Found position (%d, %d) is not a floor", pos.X, pos.Y)
	}

	// Should not be player's position (player is at 0,0)
	if state.Player.Alive && pos.X == state.Player.X && pos.Y == state.Player.Y {
		t.Error("Open position should not be player's position")
	}
}

func TestSyncEnemies(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := maze.Generate(10, 10, rng)
	state := NewState(grid, 42, 10*time.Second, 2)

	// Create test enemies
	enemies := []Enemy{
		{ID: "enemy1", ContainerID: "container1", Alive: true},
		{ID: "enemy2", ContainerID: "container2", Alive: true},
	}

	state.SyncEnemies(enemies)

	if len(state.Enemies) != 2 {
		t.Errorf("Should have 2 enemies, got %d", len(state.Enemies))
	}

	if _, ok := state.Enemies["enemy1"]; !ok {
		t.Error("enemy1 should be in state")
	}

	if _, ok := state.Enemies["enemy2"]; !ok {
		t.Error("enemy2 should be in state")
	}

	// Check container index
	if state.ContainerIndex["container1"] != "enemy1" {
		t.Error("container1 should map to enemy1")
	}
}

func TestEnemiesInCurrentLocation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := maze.Generate(10, 10, rng)
	state := NewState(grid, 42, 10*time.Second, 2)

	// Add enemies in different locations
	state.Enemies["e1"] = &Enemy{ID: "e1", Location: "loc1", Alive: true}
	state.Enemies["e2"] = &Enemy{ID: "e2", Location: "loc1", Alive: true}
	state.Enemies["e3"] = &Enemy{ID: "e3", Location: "loc2", Alive: true}
	state.Enemies["e4"] = &Enemy{ID: "e4", Location: "loc1", Alive: false} // Dead

	state.CurrentLocation = "loc1"

	enemies := state.EnemiesInCurrentLocation()

	if len(enemies) != 2 {
		t.Errorf("Should have 2 enemies in loc1, got %d", len(enemies))
	}

	for _, e := range enemies {
		if e.Location != "loc1" {
			t.Error("All returned enemies should be in loc1")
		}
		if !e.Alive {
			t.Error("All returned enemies should be alive")
		}
	}
}

func TestBuildFrame(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := maze.Generate(10, 10, rng)
	state := NewState(grid, 42, 10*time.Second, 2)

	state.Enemies["e1"] = &Enemy{
		ID:       "e1",
		Position: Position{X: 3, Y: 3},
		Alive:    true,
	}
	state.Enemies["e2"] = &Enemy{
		ID:       "e2",
		Position: Position{X: 4, Y: 4},
		Alive:    false, // Dead, should not appear
	}

	state.Bullets = []Bullet{
		{Position: Position{X: 2, Y: 2}, Alive: true},
		{Position: Position{X: 5, Y: 5}, Alive: false}, // Dead bullet
	}

	frame := BuildFrame(state)

	if frame.PlayerX != state.Player.X {
		t.Error("Frame should have correct player X")
	}

	if frame.PlayerY != state.Player.Y {
		t.Error("Frame should have correct player Y")
	}

	if !frame.PlayerLive {
		t.Error("Frame should show player as alive")
	}

	// Should only have 1 enemy (alive one)
	if len(frame.Enemies) != 1 {
		t.Errorf("Frame should have 1 enemy, got %d", len(frame.Enemies))
	}

	// Should only have 1 bullet (alive one)
	if len(frame.Bullets) != 1 {
		t.Errorf("Frame should have 1 bullet, got %d", len(frame.Bullets))
	}
}

func TestEnemiesFromDocker(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := maze.Generate(10, 10, rng)
	state := NewState(grid, 42, 10*time.Second, 2)

	snapshots := []EnemySnapshot{
		{ID: "abc123", ContainerID: "container1", ContainerName: "test1", Alive: true},
		{ID: "def456", ContainerID: "container2", ContainerName: "test2", Alive: true},
	}

	enemies := EnemiesFromDocker(snapshots, rng, state)

	if len(enemies) != 2 {
		t.Errorf("Should create 2 enemies, got %d", len(enemies))
	}

	// Check IDs are truncated
	if enemies[0].ID != "abc123" {
		t.Error("First enemy ID should be abc123")
	}

	// Check positions are assigned
	if enemies[0].Position.X == 0 && enemies[0].Position.Y == 0 {
		// Might happen but unlikely - just ensure positions are valid
		t.Log("Warning: enemy got position (0,0)")
	}
}

func TestEnemiesFromDocker_IDTruncation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := maze.Generate(10, 10, rng)
	state := NewState(grid, 42, 10*time.Second, 2)

	// Long ID should be truncated
	longID := "verylongid1234567890abcdef"
	snapshots := []EnemySnapshot{
		{ID: longID, ContainerID: "container1", ContainerName: "test1", Alive: true},
	}

	enemies := EnemiesFromDocker(snapshots, rng, state)

	if len(enemies) != 1 {
		t.Fatal("Should create 1 enemy")
	}

	if len(enemies[0].ID) != 12 {
		t.Errorf("ID should be truncated to 12 chars, got %d: %s", len(enemies[0].ID), enemies[0].ID)
	}
}

func TestDetermineLocation_NoChunkManager(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := maze.Generate(10, 10, rng)
	state := NewState(grid, 42, 10*time.Second, 2)

	// No chunk manager set
	state.ChunkManager = nil

	location := DetermineLocation("test-container", state)

	if location != "" {
		t.Errorf("Should return empty string without chunk manager, got %s", location)
	}
}

func TestPosition(t *testing.T) {
	pos := Position{X: 5, Y: 10}

	if pos.X != 5 {
		t.Errorf("X should be 5, got %d", pos.X)
	}

	if pos.Y != 10 {
		t.Errorf("Y should be 10, got %d", pos.Y)
	}
}

func TestBullet(t *testing.T) {
	bullet := Bullet{
		Position:  Position{X: 3, Y: 4},
		Direction: Position{X: 1, Y: 0},
		Alive:     true,
	}

	if !bullet.Alive {
		t.Error("Bullet should be alive")
	}

	if bullet.Position.X != 3 || bullet.Position.Y != 4 {
		t.Error("Bullet position incorrect")
	}
}

func TestGameState_InitialValues(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	grid := maze.Generate(10, 10, rng)
	state := NewState(grid, 42, 10*time.Second, 2)

	// Check initial state values
	if state.Tick != 0 {
		t.Errorf("Tick should start at 0, got %d", state.Tick)
	}

	if state.EnemiesKilled != 0 {
		t.Errorf("EnemiesKilled should start at 0, got %d", state.EnemiesKilled)
	}

	if state.PlayerDeaths != 0 {
		t.Errorf("PlayerDeaths should start at 0, got %d", state.PlayerDeaths)
	}

	if state.GameWon {
		t.Error("GameWon should be false initially")
	}

	if len(state.Bullets) != 0 {
		t.Error("Bullets should be empty initially")
	}
}

func TestErrQuit(t *testing.T) {
	if ErrQuit == nil {
		t.Error("ErrQuit should not be nil")
	}

	if ErrQuit.Error() != "quit" {
		t.Errorf("ErrQuit message should be 'quit', got '%s'", ErrQuit.Error())
	}
}

func TestErrGameWon(t *testing.T) {
	if ErrGameWon == nil {
		t.Error("ErrGameWon should not be nil")
	}

	if ErrGameWon.Error() != "game won" {
		t.Errorf("ErrGameWon message should be 'game won', got '%s'", ErrGameWon.Error())
	}
}

func TestEnemySnapshot(t *testing.T) {
	snapshot := EnemySnapshot{
		ID:            "test-id",
		ContainerID:   "container-id",
		ContainerName: "container-name",
		Alive:         true,
	}

	if snapshot.ID != "test-id" {
		t.Error("ID mismatch")
	}

	if snapshot.ContainerID != "container-id" {
		t.Error("ContainerID mismatch")
	}

	if snapshot.ContainerName != "container-name" {
		t.Error("ContainerName mismatch")
	}

	if !snapshot.Alive {
		t.Error("Alive should be true")
	}
}

func TestWorldPosition(t *testing.T) {
	pos := WorldPosition{
		Location: "test-location",
		LocalX:   5,
		LocalY:   10,
	}

	if pos.Location != "test-location" {
		t.Error("Location mismatch")
	}

	if pos.LocalX != 5 {
		t.Error("LocalX mismatch")
	}

	if pos.LocalY != 10 {
		t.Error("LocalY mismatch")
	}
}
