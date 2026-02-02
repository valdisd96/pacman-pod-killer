package ai

import (
	"fmt"
	"math"
	"math/rand"

	"pacman-pod-killer/internal/maze"
)

// Reward constants for the SARSA agent
const (
	RewardCatchPlayer  = 100.0 // Ultimate goal
	RewardMoveCloser   = 1.0   // Encourage pursuit
	RewardMoveFarther  = -0.5  // Discourage fleeing
	RewardHitWall      = -2.0  // Punish invalid moves
	RewardStepPenalty  = -0.1  // Encourage efficiency
	RewardBulletThreat = -8.0  // Strong bullet avoidance
	RewardOpenSpace    = 0.5   // Prefer escape routes
	RewardDeadEnd      = -2.0  // Avoid getting trapped
)

// Default hyperparameters
const (
	DefaultAlpha   = 0.1  // Learning rate
	DefaultGamma   = 0.95 // Discount factor
	DefaultEpsilon = 0.1  // Exploration rate (10% random actions)
)

// DecisionInfo provides detailed information about an AI decision for debugging
type DecisionInfo struct {
	EnemyID         string
	EnemyPos        struct{ X, Y int }
	StateKey        StateKey
	Action          Action
	ActionName      string
	IsExploratory   bool
	BulletsDetected [4]bool // up, down, left, right
	OpenSpaces      [4]int  // up, down, left, right
	QValues         [4]float64
	RewardBreakdown map[string]float64
	IsNewState      bool
	DodgedBullet    bool
}

// SARSA implements the SARSA on-policy reinforcement learning agent
type SARSA struct {
	qTable   *QTable
	alpha    float64 // Learning rate
	gamma    float64 // Discount factor
	epsilon  float64 // Exploration rate
	training bool    // Whether to update Q-table
	rng      *rand.Rand
	metrics  *Metrics // Performance tracking
	debug    bool     // Enable debug mode
}

// SARSAConfig holds configuration for the SARSA agent
type SARSAConfig struct {
	Alpha    float64
	Gamma    float64
	Epsilon  float64
	Training bool
	QTable   *QTable
	Metrics  *Metrics
	Debug    bool
}

// NewSARSA creates a new SARSA agent with the given configuration
func NewSARSA(cfg SARSAConfig) *SARSA {
	if cfg.Alpha == 0 {
		cfg.Alpha = DefaultAlpha
	}
	if cfg.Gamma == 0 {
		cfg.Gamma = DefaultGamma
	}
	if cfg.Epsilon == 0 {
		cfg.Epsilon = DefaultEpsilon
	}
	if cfg.QTable == nil {
		cfg.QTable = NewQTable()
	}
	if cfg.Metrics == nil {
		cfg.Metrics = NewMetrics()
	}

	return &SARSA{
		qTable:   cfg.QTable,
		alpha:    cfg.Alpha,
		gamma:    cfg.Gamma,
		epsilon:  cfg.Epsilon,
		training: cfg.Training,
		rng:      rand.New(rand.NewSource(rand.Int63())),
		metrics:  cfg.Metrics,
		debug:    cfg.Debug,
	}
}

// NewSARSADefault creates a SARSA agent with default hyperparameters
func NewSARSADefault(training bool) *SARSA {
	return NewSARSA(SARSAConfig{
		Alpha:    DefaultAlpha,
		Gamma:    DefaultGamma,
		Epsilon:  DefaultEpsilon,
		Training: training,
		QTable:   NewQTable(),
		Metrics:  NewMetrics(),
		Debug:    false,
	})
}

// SetDebug enables or disables debug mode
func (s *SARSA) SetDebug(debug bool) {
	s.debug = debug
}

// SetMetrics sets the metrics tracker
func (s *SARSA) SetMetrics(metrics *Metrics) {
	s.metrics = metrics
}

// GetMetrics returns the metrics tracker
func (s *SARSA) GetMetrics() *Metrics {
	return s.metrics
}

// SetEpsilon updates the exploration rate
func (s *SARSA) SetEpsilon(epsilon float64) {
	s.epsilon = epsilon
}

// SetTraining enables or disables training mode
func (s *SARSA) SetTraining(training bool) {
	s.training = training
}

// QTable returns the underlying Q-table (for saving/debugging)
func (s *SARSA) QTable() *QTable {
	return s.qTable
}

// ChooseAction selects an action using epsilon-greedy policy
// Returns the chosen action and whether it was exploratory
func (s *SARSA) ChooseAction(state State) (Action, bool) {
	available := state.AvailableActions()
	if len(available) == 0 {
		// No valid moves, stay in place (should rarely happen)
		return ActionUp, false
	}

	// Epsilon-greedy: explore with probability epsilon
	if s.rng.Float64() < s.epsilon {
		// Random exploration among available actions
		return available[s.rng.Intn(len(available))], true
	}

	// Greedy: choose best action
	return s.qTable.BestAction(state), false
}

// EnemyState holds the SARSA state for a single enemy (for on-policy learning)
type EnemyState struct {
	LastState    State
	LastAction   Action
	LastDistance float64
	Initialized  bool
}

// Step performs one SARSA step for an enemy and returns decision info
func (s *SARSA) Step(
	enemyX, enemyY int,
	playerX, playerY int,
	bullets []BulletInfo,
	grid *maze.Grid,
	enemyState *EnemyState,
	caughtPlayer bool,
) (dx, dy int, info DecisionInfo) {
	// Create canMove function for state computation
	canMove := func(fromX, fromY, toX, toY int) bool {
		return grid.CanMove(fromX, fromY, toX, toY)
	}

	// Count open space in each direction
	openSpaces := CountOpenSpace(grid, enemyX, enemyY)

	// Compute current state with bullet and open space info
	currentState := NewState(enemyX, enemyY, playerX, playerY, canMove, bullets, openSpaces)
	currentDistance := distance(enemyX, enemyY, playerX, playerY)
	currentKey := currentState.ToKey()

	// Get Q-values for all actions
	qVals := s.qTable.Get(currentState)

	// Choose action for current state
	action, isExploratory := s.ChooseAction(currentState)

	// Track bullet detection
	threats := DetectBulletThreat(enemyX, enemyY, bullets)

	// Determine if we dodged a bullet
	// A dodge happens when:
	// 1. There's a bullet threat at the current position
	// 2. The enemy chooses an action that moves AWAY from the threat (perpendicular or opposite)
	dodgedBullet := false
	if currentState.BulletUp && action != ActionUp {
		// Bullet coming from up, enemy chose not to go up = dodge
		dodgedBullet = true
	}
	if currentState.BulletDown && action != ActionDown {
		// Bullet coming from down, enemy chose not to go down = dodge
		dodgedBullet = true
	}
	if currentState.BulletLeft && action != ActionLeft {
		// Bullet coming from left, enemy chose not to go left = dodge
		dodgedBullet = true
	}
	if currentState.BulletRight && action != ActionRight {
		// Bullet coming from right, enemy chose not to go right = dodge
		dodgedBullet = true
	}

	// If training and we have a previous state, do SARSA update
	isNewState := false
	if s.training && enemyState.Initialized {
		reward, rewardBreakdown := s.computeRewardWithBreakdown(
			enemyState.LastDistance,
			currentDistance,
			caughtPlayer,
			enemyState.LastAction,
			enemyState.LastState,
			currentState,
		)

		// SARSA update: Q(s,a) += alpha * [r + gamma * Q(s',a') - Q(s,a)]
		nextQ := s.qTable.GetAction(currentState, action)
		target := reward + s.gamma*nextQ

		// Update and check if this was a new state
		isNewState = s.qTable.UpdateAndCheckNew(enemyState.LastState, enemyState.LastAction, s.alpha, target)

		// Record metrics
		if s.metrics != nil {
			s.metrics.RecordStateExplored(currentKey, isNewState)
			if dodgedBullet {
				s.metrics.RecordBulletDodged()
			}
		}

		// Populate decision info
		info = DecisionInfo{
			EnemyPos:        struct{ X, Y int }{enemyX, enemyY},
			StateKey:        currentKey,
			Action:          action,
			ActionName:      actionToString(action),
			IsExploratory:   isExploratory,
			BulletsDetected: threats,
			OpenSpaces:      openSpaces,
			QValues:         qVals,
			RewardBreakdown: rewardBreakdown,
			IsNewState:      isNewState,
			DodgedBullet:    dodgedBullet,
		}
	} else {
		// First step - just populate basic info
		info = DecisionInfo{
			EnemyPos:        struct{ X, Y int }{enemyX, enemyY},
			StateKey:        currentKey,
			Action:          action,
			ActionName:      actionToString(action),
			IsExploratory:   isExploratory,
			BulletsDetected: threats,
			OpenSpaces:      openSpaces,
			QValues:         qVals,
			IsNewState:      true,
			DodgedBullet:    false,
		}

		// Record initial state
		if s.metrics != nil {
			s.metrics.RecordStateExplored(currentKey, true)
		}
	}

	// Store current state/action for next step
	enemyState.LastState = currentState
	enemyState.LastAction = action
	enemyState.LastDistance = currentDistance
	enemyState.Initialized = true

	// Return movement delta and decision info
	delta := ActionDeltas[action]
	return delta[0], delta[1], info
}

func actionToString(a Action) string {
	switch a {
	case ActionUp:
		return "UP"
	case ActionDown:
		return "DOWN"
	case ActionLeft:
		return "LEFT"
	case ActionRight:
		return "RIGHT"
	}
	return "UNKNOWN"
}

// computeRewardWithBreakdown calculates reward and returns breakdown for debugging
func (s *SARSA) computeRewardWithBreakdown(
	prevDistance, currentDistance float64,
	caughtPlayer bool,
	action Action,
	prevState, nextState State,
) (float64, map[string]float64) {
	breakdown := make(map[string]float64)
	reward := RewardStepPenalty
	breakdown["step_penalty"] = RewardStepPenalty

	if caughtPlayer {
		reward += RewardCatchPlayer
		breakdown["catch_player"] = RewardCatchPlayer
		return reward, breakdown
	}

	// Check if the action was valid
	if !prevState.CanTakeAction(action) {
		reward += RewardHitWall
		breakdown["hit_wall"] = RewardHitWall
		return reward, breakdown
	}

	// Reward based on distance change
	if currentDistance < prevDistance {
		reward += RewardMoveCloser
		breakdown["move_closer"] = RewardMoveCloser
	} else if currentDistance > prevDistance {
		reward += RewardMoveFarther
		breakdown["move_farther"] = RewardMoveFarther
	}

	// Bullet threat avoidance
	bulletReward := s.computeBulletThreatReward(action, prevState)
	if bulletReward != 0 {
		reward += bulletReward
		breakdown["bullet_threat"] = bulletReward
	}

	// Open space preference
	spaceReward := s.computeOpenSpaceReward(action, prevState, nextState)
	if spaceReward != 0 {
		reward += spaceReward
		if spaceReward < 0 {
			breakdown["dead_end"] = spaceReward
		} else {
			breakdown["open_space"] = spaceReward
		}
	}

	return reward, breakdown
}

// computeReward calculates the reward for a state transition (legacy version)
func (s *SARSA) computeReward(
	prevDistance, currentDistance float64,
	caughtPlayer bool,
	action Action,
	prevState, nextState State,
) float64 {
	reward, _ := s.computeRewardWithBreakdown(prevDistance, currentDistance, caughtPlayer, action, prevState, nextState)
	return reward
}

// computeBulletThreatReward returns penalty if moving toward a bullet
func (s *SARSA) computeBulletThreatReward(action Action, state State) float64 {
	switch action {
	case ActionUp:
		if state.BulletUp {
			return RewardBulletThreat
		}
	case ActionDown:
		if state.BulletDown {
			return RewardBulletThreat
		}
	case ActionLeft:
		if state.BulletLeft {
			return RewardBulletThreat
		}
	case ActionRight:
		if state.BulletRight {
			return RewardBulletThreat
		}
	}
	return 0
}

// computeOpenSpaceReward rewards moving toward more open space
func (s *SARSA) computeOpenSpaceReward(action Action, prevState, nextState State) float64 {
	reward := 0.0

	switch action {
	case ActionUp:
		if nextState.OpenUp == 0 {
			reward += RewardDeadEnd
		} else {
			reward += float64(nextState.OpenUp) * RewardOpenSpace / 3.0
		}
	case ActionDown:
		if nextState.OpenDown == 0 {
			reward += RewardDeadEnd
		} else {
			reward += float64(nextState.OpenDown) * RewardOpenSpace / 3.0
		}
	case ActionLeft:
		if nextState.OpenLeft == 0 {
			reward += RewardDeadEnd
		} else {
			reward += float64(nextState.OpenLeft) * RewardOpenSpace / 3.0
		}
	case ActionRight:
		if nextState.OpenRight == 0 {
			reward += RewardDeadEnd
		} else {
			reward += float64(nextState.OpenRight) * RewardOpenSpace / 3.0
		}
	}

	return reward
}

// distance computes Euclidean distance between two points
func distance(x1, y1, x2, y2 int) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	return math.Sqrt(dx*dx + dy*dy)
}

// NextStep implements a simple interface compatible with how Random AI is called
func (s *SARSA) NextStep(grid *maze.Grid, enemyX, enemyY, playerX, playerY int) (int, int) {
	canMove := func(fromX, fromY, toX, toY int) bool {
		return grid.CanMove(fromX, fromY, toX, toY)
	}

	openSpaces := CountOpenSpace(grid, enemyX, enemyY)
	var emptyBullets []BulletInfo

	state := NewState(enemyX, enemyY, playerX, playerY, canMove, emptyBullets, openSpaces)
	action, _ := s.ChooseAction(state)

	if !state.CanTakeAction(action) {
		available := state.AvailableActions()
		if len(available) > 0 {
			action = available[0]
		} else {
			return 0, 0
		}
	}

	delta := ActionDeltas[action]
	return delta[0], delta[1]
}

// HandlePlayerCaught should be called when an enemy catches the player
func (s *SARSA) HandlePlayerCaught(enemyState *EnemyState, grid *maze.Grid, enemyX, enemyY, playerX, playerY int) {
	if !s.training || !enemyState.Initialized {
		return
	}

	canMove := func(fromX, fromY, toX, toY int) bool {
		return grid.CanMove(fromX, fromY, toX, toY)
	}

	openSpaces := CountOpenSpace(grid, enemyX, enemyY)
	var emptyBullets []BulletInfo

	currentState := NewState(enemyX, enemyY, playerX, playerY, canMove, emptyBullets, openSpaces)
	action, _ := s.ChooseAction(currentState)

	reward := s.computeReward(
		enemyState.LastDistance,
		0,
		true,
		enemyState.LastAction,
		enemyState.LastState,
		currentState,
	)

	target := reward + s.gamma*0
	s.qTable.Update(enemyState.LastState, enemyState.LastAction, s.alpha, target)
	s.qTable.Update(currentState, action, s.alpha, RewardCatchPlayer)
}

// Stats returns statistics about the Q-table for debugging
func (s *SARSA) Stats() (numStates int, avgQValue float64) {
	numStates = s.qTable.Size()
	if numStates == 0 {
		return 0, 0
	}
	return numStates, 0
}

// FormatDecisionInfo returns a human-readable string of decision info for logging
func FormatDecisionInfo(info DecisionInfo) string {
	return fmt.Sprintf(
		"Enemy at (%d,%d) state=%d action=%s new=%v dodged=%v bullets=[%v,%v,%v,%v] reward=%+v",
		info.EnemyPos.X, info.EnemyPos.Y,
		info.StateKey,
		info.ActionName,
		info.IsNewState,
		info.DodgedBullet,
		info.BulletsDetected[0], info.BulletsDetected[1], info.BulletsDetected[2], info.BulletsDetected[3],
		info.RewardBreakdown,
	)
}
