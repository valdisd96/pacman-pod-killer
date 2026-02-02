package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Metrics tracks AI performance and learning statistics
type Metrics struct {
	mu sync.RWMutex

	// Persistent metrics (saved to file)
	TotalStatesExplored int   `json:"total_states_explored"` // Unique states ever seen across all sessions
	TotalBulletsDodged  int   `json:"total_bullets_dodged"`  // Total bullets successfully avoided
	TotalGamesPlayed    int   `json:"total_games_played"`    // Number of games/sessions
	SessionStartStates  int   `json:"session_start_states"`  // States known at session start
	SessionStartTime    int64 `json:"session_start_time"`    // Unix timestamp

	// Current session metrics (volatile)
	SessionStatesExplored int   `json:"-"` // New states this session
	SessionBulletsDodged  int   `json:"-"` // Bullets dodged this session
	CurrentTick           int64 `json:"-"` // Current game tick
}

// DefaultMetricsPath returns the default path for metrics file
func DefaultMetricsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pacman-pod-killer/metrics.json"
	}
	return filepath.Join(home, ".pacman-pod-killer", "metrics.json")
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		SessionStartTime: 0, // Will be set on Load
	}
}

// LoadMetrics loads metrics from file or creates new if doesn't exist
func LoadMetrics(path string) (*Metrics, error) {
	m := NewMetrics()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, start fresh
			return m, nil
		}
		return nil, fmt.Errorf("failed to read metrics file: %w", err)
	}

	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("failed to decode metrics: %w", err)
	}

	// Remember where we started this session
	m.SessionStartStates = m.TotalStatesExplored
	m.SessionStatesExplored = 0
	m.SessionBulletsDodged = 0

	return m, nil
}

// Save persists metrics to file
func (m *Metrics) Save(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode metrics: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	return nil
}

// RecordStateExplored increments the states explored counter
func (m *Metrics) RecordStateExplored(stateKey StateKey, isNew bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if isNew {
		m.TotalStatesExplored++
		m.SessionStatesExplored++
	}
}

// RecordBulletDodged increments the bullet dodge counter
func (m *Metrics) RecordBulletDodged() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalBulletsDodged++
	m.SessionBulletsDodged++
}

// RecordGameStart increments games played counter
func (m *Metrics) RecordGameStart() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalGamesPlayed++
}

// SetCurrentTick updates the current tick for real-time display
func (m *Metrics) SetCurrentTick(tick int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CurrentTick = tick
}

// GetExplorationStats returns (states_explored, total_possible_states, percentage)
// Total possible states is approximately 1.6M (5*5 * 16 * 16 * 256)
const TotalPossibleStates = 1638400 // 5*5 * 16 * 16 * 256

func (m *Metrics) GetExplorationStats() (explored int, total int, percentage float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.TotalStatesExplored, TotalPossibleStates, float64(m.TotalStatesExplored) / float64(TotalPossibleStates) * 100
}

// GetDisplayString returns the status bar format
// Example: "AI:12.5% dodged:45" or "AI:0.0061% dodged:45" (exploration % + bullets dodged)
// Shows 4 decimal places when percentage is small to give meaningful feedback during learning
func (m *Metrics) GetDisplayString() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, _, percentage := m.GetExplorationStats()

	// Use adaptive precision: show 1 decimal for >=0.1%, 4 decimals for smaller values
	if percentage >= 0.1 {
		return fmt.Sprintf("AI:%.1f%% dodged:%d", percentage, m.TotalBulletsDodged)
	} else if percentage >= 0.0001 {
		return fmt.Sprintf("AI:%.4f%% dodged:%d", percentage, m.TotalBulletsDodged)
	} else {
		// Even smaller - show raw count to give user feedback
		return fmt.Sprintf("AI:%d/%d dodged:%d", m.TotalStatesExplored, TotalPossibleStates, m.TotalBulletsDodged)
	}
}

// GetSessionStats returns current session statistics for logging
func (m *Metrics) GetSessionStats() (newStates int, sessionDodges int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.SessionStatesExplored, m.SessionBulletsDodged
}

// ResetSession resets session-specific counters
func (m *Metrics) ResetSession() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SessionStatesExplored = 0
	m.SessionBulletsDodged = 0
	m.SessionStartStates = m.TotalStatesExplored
}
