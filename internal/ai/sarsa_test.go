package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateKey(t *testing.T) {
	// Test that different states produce different keys
	s1 := State{DeltaX: 0, DeltaY: 0, CanUp: true, CanDown: true, CanLeft: true, CanRight: true}
	s2 := State{DeltaX: 1, DeltaY: 0, CanUp: true, CanDown: true, CanLeft: true, CanRight: true}
	s3 := State{DeltaX: 0, DeltaY: 1, CanUp: true, CanDown: true, CanLeft: true, CanRight: true}
	s4 := State{DeltaX: 0, DeltaY: 0, CanUp: false, CanDown: true, CanLeft: true, CanRight: true}

	if s1.ToKey() == s2.ToKey() {
		t.Error("Different DeltaX should produce different keys")
	}
	if s1.ToKey() == s3.ToKey() {
		t.Error("Different DeltaY should produce different keys")
	}
	if s1.ToKey() == s4.ToKey() {
		t.Error("Different CanUp should produce different keys")
	}
}

func TestStateKeyConsistency(t *testing.T) {
	// Test that the same state always produces the same key
	s := State{DeltaX: -2, DeltaY: 2, CanUp: true, CanDown: false, CanLeft: true, CanRight: false}
	key1 := s.ToKey()
	key2 := s.ToKey()

	if key1 != key2 {
		t.Errorf("Same state should produce same key, got %d and %d", key1, key2)
	}
}

func TestClampDelta(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{-5, -2},
		{-2, -2},
		{-1, -1},
		{0, 0},
		{1, 1},
		{2, 2},
		{5, 2},
	}

	for _, tt := range tests {
		result := clampDelta(tt.input)
		if result != tt.expected {
			t.Errorf("clampDelta(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestAvailableActions(t *testing.T) {
	s := State{CanUp: true, CanDown: false, CanLeft: true, CanRight: false}
	actions := s.AvailableActions()

	if len(actions) != 2 {
		t.Errorf("Expected 2 available actions, got %d", len(actions))
	}

	found := make(map[Action]bool)
	for _, a := range actions {
		found[a] = true
	}

	if !found[ActionUp] {
		t.Error("Expected ActionUp to be available")
	}
	if !found[ActionLeft] {
		t.Error("Expected ActionLeft to be available")
	}
	if found[ActionDown] {
		t.Error("ActionDown should not be available")
	}
	if found[ActionRight] {
		t.Error("ActionRight should not be available")
	}
}

func TestCanTakeAction(t *testing.T) {
	s := State{CanUp: true, CanDown: false, CanLeft: true, CanRight: false}

	if !s.CanTakeAction(ActionUp) {
		t.Error("Should be able to take ActionUp")
	}
	if s.CanTakeAction(ActionDown) {
		t.Error("Should not be able to take ActionDown")
	}
	if !s.CanTakeAction(ActionLeft) {
		t.Error("Should be able to take ActionLeft")
	}
	if s.CanTakeAction(ActionRight) {
		t.Error("Should not be able to take ActionRight")
	}
}

func TestQTableBasicOperations(t *testing.T) {
	q := NewQTable()

	s := State{DeltaX: 1, DeltaY: 1, CanUp: true, CanDown: true, CanLeft: true, CanRight: true}

	// Initially all values should be zero
	vals := q.Get(s)
	for i, v := range vals {
		if v != 0 {
			t.Errorf("Expected initial value 0 for action %d, got %f", i, v)
		}
	}

	// Set a value
	q.Set(s, ActionUp, 10.0)
	if q.GetAction(s, ActionUp) != 10.0 {
		t.Errorf("Expected 10.0, got %f", q.GetAction(s, ActionUp))
	}

	// Other actions should still be zero
	if q.GetAction(s, ActionDown) != 0 {
		t.Errorf("Expected 0, got %f", q.GetAction(s, ActionDown))
	}
}

func TestQTableUpdate(t *testing.T) {
	q := NewQTable()
	s := State{DeltaX: 0, DeltaY: 0, CanUp: true, CanDown: true, CanLeft: true, CanRight: true}

	// Q(s,a) = 0 initially
	// Update with target=10, alpha=0.1
	// Q(s,a) += 0.1 * (10 - 0) = 1.0
	q.Update(s, ActionUp, 0.1, 10.0)

	expected := 1.0
	actual := q.GetAction(s, ActionUp)
	if actual != expected {
		t.Errorf("Expected %f after update, got %f", expected, actual)
	}

	// Update again with target=10
	// Q(s,a) += 0.1 * (10 - 1) = 1 + 0.9 = 1.9
	q.Update(s, ActionUp, 0.1, 10.0)
	expected = 1.9
	actual = q.GetAction(s, ActionUp)
	if actual != expected {
		t.Errorf("Expected %f after second update, got %f", expected, actual)
	}
}

func TestQTableSaveLoad(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "qtable_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "test_qtable.json")

	// Create and populate Q-table
	q1 := NewQTable()
	s1 := State{DeltaX: 1, DeltaY: -1, CanUp: true, CanDown: false, CanLeft: true, CanRight: false}
	s2 := State{DeltaX: 2, DeltaY: 2, CanUp: true, CanDown: true, CanLeft: true, CanRight: true}

	q1.Set(s1, ActionUp, 5.0)
	q1.Set(s1, ActionLeft, 3.0)
	q1.Set(s2, ActionDown, 7.5)

	// Save
	if err := q1.Save(path); err != nil {
		t.Fatalf("Failed to save Q-table: %v", err)
	}

	// Load
	q2, err := LoadQTable(path)
	if err != nil {
		t.Fatalf("Failed to load Q-table: %v", err)
	}

	// Verify values
	if q2.GetAction(s1, ActionUp) != 5.0 {
		t.Errorf("Expected 5.0 for s1/ActionUp, got %f", q2.GetAction(s1, ActionUp))
	}
	if q2.GetAction(s1, ActionLeft) != 3.0 {
		t.Errorf("Expected 3.0 for s1/ActionLeft, got %f", q2.GetAction(s1, ActionLeft))
	}
	if q2.GetAction(s2, ActionDown) != 7.5 {
		t.Errorf("Expected 7.5 for s2/ActionDown, got %f", q2.GetAction(s2, ActionDown))
	}
}

func TestLoadNonexistentQTable(t *testing.T) {
	q, err := LoadQTable("/nonexistent/path/qtable.json")
	if err != nil {
		t.Fatalf("Loading nonexistent file should not error, got: %v", err)
	}
	if q.Size() != 0 {
		t.Errorf("Loaded Q-table should be empty, got %d states", q.Size())
	}
}

func TestSARSAChooseAction(t *testing.T) {
	sarsa := NewSARSADefault(false)
	sarsa.SetEpsilon(0) // No exploration, always greedy

	s := State{DeltaX: 1, DeltaY: 0, CanUp: true, CanDown: true, CanLeft: true, CanRight: true}

	// Set ActionRight as best action
	sarsa.QTable().Set(s, ActionRight, 10.0)
	sarsa.QTable().Set(s, ActionUp, 1.0)

	// Should always choose ActionRight when epsilon=0
	action, wasRandom := sarsa.ChooseAction(s)
	if wasRandom {
		t.Error("Action should not be random when epsilon=0")
	}
	if action != ActionRight {
		t.Errorf("Expected ActionRight, got %d", action)
	}
}

func TestSARSAExploration(t *testing.T) {
	sarsa := NewSARSADefault(false)
	sarsa.SetEpsilon(1.0) // Always explore

	s := State{DeltaX: 0, DeltaY: 0, CanUp: true, CanDown: true, CanLeft: true, CanRight: true}
	sarsa.QTable().Set(s, ActionRight, 100.0) // Even with high Q-value

	// With epsilon=1, should always be random
	_, wasRandom := sarsa.ChooseAction(s)
	if !wasRandom {
		t.Error("Action should be random when epsilon=1")
	}
}

func TestBestAction(t *testing.T) {
	q := NewQTable()
	s := State{DeltaX: 0, DeltaY: 0, CanUp: true, CanDown: true, CanLeft: false, CanRight: true}

	q.Set(s, ActionDown, 5.0)
	q.Set(s, ActionUp, 3.0)
	q.Set(s, ActionRight, 1.0)
	q.Set(s, ActionLeft, 10.0) // Highest but not available

	best := q.BestAction(s)
	if best != ActionDown {
		t.Errorf("Expected ActionDown (best available), got %d", best)
	}
}

func TestActionDeltas(t *testing.T) {
	// Verify action deltas are correct
	if ActionDeltas[ActionUp][0] != 0 || ActionDeltas[ActionUp][1] != -1 {
		t.Error("ActionUp delta should be (0, -1)")
	}
	if ActionDeltas[ActionDown][0] != 0 || ActionDeltas[ActionDown][1] != 1 {
		t.Error("ActionDown delta should be (0, 1)")
	}
	if ActionDeltas[ActionLeft][0] != -1 || ActionDeltas[ActionLeft][1] != 0 {
		t.Error("ActionLeft delta should be (-1, 0)")
	}
	if ActionDeltas[ActionRight][0] != 1 || ActionDeltas[ActionRight][1] != 0 {
		t.Error("ActionRight delta should be (1, 0)")
	}
}
