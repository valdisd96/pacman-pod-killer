package input

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParseEvent_Quit(t *testing.T) {
	tests := []struct {
		name string
		key  tcell.Key
		want Action
	}{
		{"Escape key", tcell.KeyEscape, ActionQuit},
		{"Ctrl+C", tcell.KeyCtrlC, ActionQuit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tcell.NewEventKey(tt.key, 0, tcell.ModNone)
			got := parseEvent(event)
			if got != tt.want {
				t.Errorf("parseEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEvent_ArrowKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tcell.Key
		want Action
	}{
		{"Up arrow", tcell.KeyUp, ActionUp},
		{"Down arrow", tcell.KeyDown, ActionDown},
		{"Left arrow", tcell.KeyLeft, ActionLeft},
		{"Right arrow", tcell.KeyRight, ActionRight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tcell.NewEventKey(tt.key, 0, tcell.ModNone)
			got := parseEvent(event)
			if got != tt.want {
				t.Errorf("parseEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEvent_WASDKeys(t *testing.T) {
	tests := []struct {
		name     string
		rune     rune
		expected Action
	}{
		{"W key", 'w', ActionUp},
		{"W key uppercase", 'W', ActionUp},
		{"S key", 's', ActionDown},
		{"S key uppercase", 'S', ActionDown},
		{"A key", 'a', ActionLeft},
		{"A key uppercase", 'A', ActionLeft},
		{"D key", 'd', ActionRight},
		{"D key uppercase", 'D', ActionRight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tcell.NewEventKey(tcell.KeyRune, tt.rune, tcell.ModNone)
			got := parseEvent(event)
			if got != tt.expected {
				t.Errorf("parseEvent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseEvent_Shoot(t *testing.T) {
	event := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	got := parseEvent(event)
	if got != ActionShoot {
		t.Errorf("parseEvent() for space = %v, want %v", got, ActionShoot)
	}
}

func TestParseEvent_QuitKeys(t *testing.T) {
	tests := []struct {
		name     string
		rune     rune
		expected Action
	}{
		{"Q key", 'q', ActionQuit},
		{"Q key uppercase", 'Q', ActionQuit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tcell.NewEventKey(tcell.KeyRune, tt.rune, tcell.ModNone)
			got := parseEvent(event)
			if got != tt.expected {
				t.Errorf("parseEvent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseEvent_InvalidKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tcell.Key
		r    rune
	}{
		{"Enter key", tcell.KeyEnter, 0},
		{"Tab key", tcell.KeyTab, 0},
		{"Backspace", tcell.KeyBackspace, 0},
		{"Unknown rune", tcell.KeyRune, 'x'},
		{"Unknown rune uppercase", tcell.KeyRune, 'Z'},
		{"Number key", tcell.KeyRune, '1'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tcell.NewEventKey(tt.key, tt.r, tcell.ModNone)
			got := parseEvent(event)
			if got != ActionNone {
				t.Errorf("parseEvent() = %v, want %v", got, ActionNone)
			}
		})
	}
}

func TestParseEvent_NonKeyEvent(t *testing.T) {
	// Create a mock resize event (should return ActionNone)
	event := tcell.NewEventResize(10, 20)
	got := parseEvent(event)
	if got != ActionNone {
		t.Errorf("parseEvent() for resize event = %v, want %v", got, ActionNone)
	}
}

func TestAction_Constants(t *testing.T) {
	// Verify action values are as expected
	if ActionNone != 0 {
		t.Error("ActionNone should be 0")
	}
	if ActionUp != 1 {
		t.Error("ActionUp should be 1")
	}
	if ActionDown != 2 {
		t.Error("ActionDown should be 2")
	}
	if ActionLeft != 3 {
		t.Error("ActionLeft should be 3")
	}
	if ActionRight != 4 {
		t.Error("ActionRight should be 4")
	}
	if ActionQuit != 5 {
		t.Error("ActionQuit should be 5")
	}
	if ActionShoot != 6 {
		t.Error("ActionShoot should be 6")
	}
}
