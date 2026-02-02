package ai

// Action represents a movement direction for the SARSA agent
type Action int

const (
	ActionUp    Action = 0
	ActionDown  Action = 1
	ActionLeft  Action = 2
	ActionRight Action = 3
	NumActions         = 4
)

// ActionDeltas maps actions to (dx, dy) movement deltas
var ActionDeltas = [NumActions][2]int{
	ActionUp:    {0, -1},
	ActionDown:  {0, 1},
	ActionLeft:  {-1, 0},
	ActionRight: {1, 0},
}

// State represents what the enemy "sees" for SARSA learning
// This is discretized to keep the Q-table manageable
// State space: 5*5 * 16 * 16 * 256 = ~1.6M states (was ~400)
type State struct {
	// Relative position to player (clamped to range -2 to +2)
	DeltaX int
	DeltaY int

	// Available moves (walls around enemy)
	CanUp    bool
	CanDown  bool
	CanLeft  bool
	CanRight bool

	// Bullet threats (bullet approaching from each direction within 7 cells)
	BulletUp    bool
	BulletDown  bool
	BulletLeft  bool
	BulletRight bool

	// Open space in each direction (0-3 cells, clamped)
	OpenUp    int
	OpenDown  int
	OpenLeft  int
	OpenRight int
}

// StateKey is a compact representation of State for use as a map key
// Format: encodes all state components into a single int64
type StateKey int64

// ToKey converts a State to a compact StateKey for Q-table lookup
func (s State) ToKey() StateKey {
	// DeltaX and DeltaY are in range [-2, 2], so we shift to [0, 4] (5 values each)
	dx := s.DeltaX + 2
	dy := s.DeltaY + 2

	// Clamp to valid range
	if dx < 0 {
		dx = 0
	}
	if dx > 4 {
		dx = 4
	}
	if dy < 0 {
		dy = 0
	}
	if dy > 4 {
		dy = 4
	}

	// Pack order: dx, dy, Can[4], Bullet[4], Open[4]
	// dx: 5 values (3 bits)
	// dy: 5 values (3 bits)
	// CanUp/Down/Left/Right: 4 booleans (4 bits)
	// BulletUp/Down/Left/Right: 4 booleans (4 bits)
	// OpenUp/Down/Left/Right: 4 values 0-3, 2 bits each (8 bits)
	// Total: 5 * 5 * 16 * 16 * 4^4 = 5 * 5 * 16 * 16 * 256 = ~1.6M states

	key := int64(dx)
	key = key*5 + int64(dy)

	// Pack wall sensors (4 bits)
	key = key*2 + boolToInt(s.CanUp)
	key = key*2 + boolToInt(s.CanDown)
	key = key*2 + boolToInt(s.CanLeft)
	key = key*2 + boolToInt(s.CanRight)

	// Pack bullet threats (4 bits)
	key = key*2 + boolToInt(s.BulletUp)
	key = key*2 + boolToInt(s.BulletDown)
	key = key*2 + boolToInt(s.BulletLeft)
	key = key*2 + boolToInt(s.BulletRight)

	// Pack open space (4 values of 0-3, 2 bits each = 8 bits)
	key = key*4 + int64(clampTo3(s.OpenUp))
	key = key*4 + int64(clampTo3(s.OpenDown))
	key = key*4 + int64(clampTo3(s.OpenLeft))
	key = key*4 + int64(clampTo3(s.OpenRight))

	return StateKey(key)
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func clampTo3(v int) int {
	if v < 0 {
		return 0
	}
	if v > 3 {
		return 3
	}
	return v
}

// clampDelta clamps a value to the range [-2, 2]
func clampDelta(v int) int {
	if v < -2 {
		return -2
	}
	if v > 2 {
		return 2
	}
	return v
}

// NewState creates a State from enemy and player positions plus maze info
func NewState(enemyX, enemyY, playerX, playerY int, canMove func(fromX, fromY, toX, toY int) bool, bullets []BulletInfo, openSpaces [4]int) State {
	// Detect bullet threats
	threats := DetectBulletThreat(enemyX, enemyY, bullets)

	return State{
		DeltaX:      clampDelta(playerX - enemyX),
		DeltaY:      clampDelta(playerY - enemyY),
		CanUp:       canMove(enemyX, enemyY, enemyX, enemyY-1),
		CanDown:     canMove(enemyX, enemyY, enemyX, enemyY+1),
		CanLeft:     canMove(enemyX, enemyY, enemyX-1, enemyY),
		CanRight:    canMove(enemyX, enemyY, enemyX+1, enemyY),
		BulletUp:    threats[0],
		BulletDown:  threats[1],
		BulletLeft:  threats[2],
		BulletRight: threats[3],
		OpenUp:      openSpaces[0],
		OpenDown:    openSpaces[1],
		OpenLeft:    openSpaces[2],
		OpenRight:   openSpaces[3],
	}
}

// AvailableActions returns a slice of actions that are valid from this state
func (s State) AvailableActions() []Action {
	actions := make([]Action, 0, 4)
	if s.CanUp {
		actions = append(actions, ActionUp)
	}
	if s.CanDown {
		actions = append(actions, ActionDown)
	}
	if s.CanLeft {
		actions = append(actions, ActionLeft)
	}
	if s.CanRight {
		actions = append(actions, ActionRight)
	}
	return actions
}

// CanTakeAction checks if a specific action is valid from this state
func (s State) CanTakeAction(a Action) bool {
	switch a {
	case ActionUp:
		return s.CanUp
	case ActionDown:
		return s.CanDown
	case ActionLeft:
		return s.CanLeft
	case ActionRight:
		return s.CanRight
	default:
		return false
	}
}
