package ai

import (
	"math"
	"math/rand"

	"pacman-pod-killer/internal/maze"
)

// Reward constants for the SARSA agent
const (
	RewardCatchPlayer = 100.0 // Ultimate goal
	RewardMoveCloser  = 1.0   // Encourage pursuit
	RewardMoveFarther = -0.5  // Discourage fleeing
	RewardHitWall     = -2.0  // Punish invalid moves
	RewardStepPenalty = -0.1  // Encourage efficiency
)

// Default hyperparameters
const (
	DefaultAlpha   = 0.1  // Learning rate
	DefaultGamma   = 0.95 // Discount factor
	DefaultEpsilon = 0.1  // Exploration rate (10% random actions)
)

// SARSA implements the SARSA on-policy reinforcement learning agent
type SARSA struct {
	qTable   *QTable
	alpha    float64 // Learning rate
	gamma    float64 // Discount factor
	epsilon  float64 // Exploration rate
	training bool    // Whether to update Q-table
	rng      *rand.Rand
}

// SARSAConfig holds configuration for the SARSA agent
type SARSAConfig struct {
	Alpha    float64
	Gamma    float64
	Epsilon  float64
	Training bool
	QTable   *QTable
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

	return &SARSA{
		qTable:   cfg.QTable,
		alpha:    cfg.Alpha,
		gamma:    cfg.Gamma,
		epsilon:  cfg.Epsilon,
		training: cfg.Training,
		rng:      rand.New(rand.NewSource(rand.Int63())),
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
	})
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

// Step performs one SARSA step for an enemy:
// 1. Observe current state
// 2. Choose action (epsilon-greedy)
// 3. If this is not the first step, update Q-value for the previous state-action
// 4. Return the action to take
func (s *SARSA) Step(
	enemyX, enemyY int,
	playerX, playerY int,
	grid *maze.Grid,
	enemyState *EnemyState,
	caughtPlayer bool,
) (dx, dy int) {
	// Create canMove function for state computation
	canMove := func(fromX, fromY, toX, toY int) bool {
		return grid.CanMove(fromX, fromY, toX, toY)
	}

	// Compute current state
	currentState := NewState(enemyX, enemyY, playerX, playerY, canMove)
	currentDistance := distance(enemyX, enemyY, playerX, playerY)

	// Choose action for current state
	action, _ := s.ChooseAction(currentState)

	// If training and we have a previous state, do SARSA update
	if s.training && enemyState.Initialized {
		reward := s.computeReward(
			enemyState.LastDistance,
			currentDistance,
			caughtPlayer,
			enemyState.LastAction,
			enemyState.LastState,
		)

		// SARSA update: Q(s,a) += alpha * [r + gamma * Q(s',a') - Q(s,a)]
		nextQ := s.qTable.GetAction(currentState, action)
		target := reward + s.gamma*nextQ
		s.qTable.Update(enemyState.LastState, enemyState.LastAction, s.alpha, target)
	}

	// Store current state/action for next step
	enemyState.LastState = currentState
	enemyState.LastAction = action
	enemyState.LastDistance = currentDistance
	enemyState.Initialized = true

	// Return movement delta
	delta := ActionDeltas[action]
	return delta[0], delta[1]
}

// computeReward calculates the reward for a state transition
func (s *SARSA) computeReward(
	prevDistance, currentDistance float64,
	caughtPlayer bool,
	action Action,
	state State,
) float64 {
	reward := RewardStepPenalty // Base step penalty

	if caughtPlayer {
		reward += RewardCatchPlayer
		return reward
	}

	// Check if the action was valid
	if !state.CanTakeAction(action) {
		reward += RewardHitWall
		return reward
	}

	// Reward based on distance change
	if currentDistance < prevDistance {
		reward += RewardMoveCloser
	} else if currentDistance > prevDistance {
		reward += RewardMoveFarther
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
// This is a stateless call that doesn't do learning - use Step() for full SARSA
func (s *SARSA) NextStep(grid *maze.Grid, enemyX, enemyY, playerX, playerY int) (int, int) {
	canMove := func(fromX, fromY, toX, toY int) bool {
		return grid.CanMove(fromX, fromY, toX, toY)
	}

	state := NewState(enemyX, enemyY, playerX, playerY, canMove)
	action, _ := s.ChooseAction(state)

	// Validate action is possible
	if !state.CanTakeAction(action) {
		// Fall back to any available action
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
// This gives the final reward for the catching enemy
func (s *SARSA) HandlePlayerCaught(enemyState *EnemyState, grid *maze.Grid, enemyX, enemyY, playerX, playerY int) {
	if !s.training || !enemyState.Initialized {
		return
	}

	canMove := func(fromX, fromY, toX, toY int) bool {
		return grid.CanMove(fromX, fromY, toX, toY)
	}

	// Compute terminal state
	currentState := NewState(enemyX, enemyY, playerX, playerY, canMove)
	action, _ := s.ChooseAction(currentState)

	// Final SARSA update with catch reward
	reward := s.computeReward(
		enemyState.LastDistance,
		0, // Distance is 0 when caught
		true,
		enemyState.LastAction,
		enemyState.LastState,
	)

	// Terminal state has Q=0 for next state
	target := reward + s.gamma*0
	s.qTable.Update(enemyState.LastState, enemyState.LastAction, s.alpha, target)

	// Also give a small update to the current state for reaching the goal
	s.qTable.Update(currentState, action, s.alpha, RewardCatchPlayer)
}

// Stats returns statistics about the Q-table for debugging
func (s *SARSA) Stats() (numStates int, avgQValue float64) {
	numStates = s.qTable.Size()
	if numStates == 0 {
		return 0, 0
	}

	// This is a simple stat, could be expanded
	return numStates, 0
}
