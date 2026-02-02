package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// QTable stores Q-values for state-action pairs
// Thread-safe for concurrent access during gameplay
type QTable struct {
	mu     sync.RWMutex
	values map[StateKey][NumActions]float64
}

// NewQTable creates a new empty Q-table
func NewQTable() *QTable {
	return &QTable{
		values: make(map[StateKey][NumActions]float64),
	}
}

// Get returns Q-values for all actions in a given state
func (q *QTable) Get(s State) [NumActions]float64 {
	q.mu.RLock()
	defer q.mu.RUnlock()

	key := s.ToKey()
	if vals, exists := q.values[key]; exists {
		return vals
	}
	// Return zeros for unseen states (optimistic initialization could be used here)
	return [NumActions]float64{}
}

// GetAction returns the Q-value for a specific state-action pair
func (q *QTable) GetAction(s State, a Action) float64 {
	q.mu.RLock()
	defer q.mu.RUnlock()

	key := s.ToKey()
	if vals, exists := q.values[key]; exists {
		return vals[a]
	}
	return 0.0
}

// Set updates the Q-value for a specific state-action pair
func (q *QTable) Set(s State, a Action, value float64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	key := s.ToKey()
	vals := q.values[key]
	vals[a] = value
	q.values[key] = vals
}

// Update applies the SARSA update rule: Q(s,a) += alpha * (target - Q(s,a))
func (q *QTable) Update(s State, a Action, alpha, target float64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	key := s.ToKey()
	vals := q.values[key]
	vals[a] += alpha * (target - vals[a])
	q.values[key] = vals
}

// UpdateAndCheckNew applies SARSA update and returns true if state was newly discovered
func (q *QTable) UpdateAndCheckNew(s State, a Action, alpha, target float64) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	key := s.ToKey()
	_, existed := q.values[key]
	vals := q.values[key]
	vals[a] += alpha * (target - vals[a])
	q.values[key] = vals

	return !existed
}

// Size returns the number of states in the Q-table
func (q *QTable) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.values)
}

// QTableVersion is the current version of the Q-table format
// Version 1: Original state (400 states) - player position + wall sensors
// Version 2: Enhanced state (~1.6M states) - adds bullet threats + open space
const QTableVersion = "2.0"

// QTableJSON is the serializable format for the Q-table
type QTableJSON struct {
	Version string                         `json:"version"`
	States  map[string][NumActions]float64 `json:"states"`
}

// Save writes the Q-table to a JSON file
func (q *QTable) Save(path string) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Convert to serializable format
	data := QTableJSON{
		Version: QTableVersion,
		States:  make(map[string][NumActions]float64),
	}

	for key, vals := range q.values {
		data.States[fmt.Sprintf("%d", key)] = vals
	}

	// Write to file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode Q-table: %w", err)
	}

	return nil
}

// LoadQTable loads a Q-table from a JSON file
// Returns an empty Q-table if the file doesn't exist
func LoadQTable(path string) (*QTable, error) {
	q := NewQTable()

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty Q-table
			return q, nil
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	var data QTableJSON
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode Q-table: %w", err)
	}

	// Check version - start fresh if version mismatch
	if data.Version != QTableVersion {
		// Version mismatch, return empty Q-table
		// Old state format is incompatible with new features
		return q, nil
	}

	// Convert from serializable format
	for keyStr, vals := range data.States {
		var key int64
		if _, err := fmt.Sscanf(keyStr, "%d", &key); err != nil {
			continue // Skip invalid keys
		}
		q.values[StateKey(key)] = vals
	}

	return q, nil
}

// DefaultQTablePath returns the default path for the Q-table file
func DefaultQTablePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pacman-pod-killer/qtable.json"
	}
	return filepath.Join(home, ".pacman-pod-killer", "qtable.json")
}

// MaxQValue returns the maximum Q-value for available actions in a state
func (q *QTable) MaxQValue(s State) float64 {
	q.mu.RLock()
	defer q.mu.RUnlock()

	key := s.ToKey()
	vals, exists := q.values[key]
	if !exists {
		return 0.0
	}

	actions := s.AvailableActions()
	if len(actions) == 0 {
		return 0.0
	}

	maxQ := vals[actions[0]]
	for _, a := range actions[1:] {
		if vals[a] > maxQ {
			maxQ = vals[a]
		}
	}
	return maxQ
}

// BestAction returns the action with the highest Q-value for available actions
// Returns ActionUp as default if no actions available
func (q *QTable) BestAction(s State) Action {
	q.mu.RLock()
	defer q.mu.RUnlock()

	actions := s.AvailableActions()
	if len(actions) == 0 {
		return ActionUp
	}

	key := s.ToKey()
	vals, exists := q.values[key]
	if !exists {
		// No learned values, return first available action
		return actions[0]
	}

	bestAction := actions[0]
	bestQ := vals[actions[0]]
	for _, a := range actions[1:] {
		if vals[a] > bestQ {
			bestQ = vals[a]
			bestAction = a
		}
	}
	return bestAction
}
