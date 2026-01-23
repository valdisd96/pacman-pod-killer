package input

import "github.com/gdamore/tcell/v2"

type Action int

const (
	ActionNone Action = iota
	ActionUp
	ActionDown
	ActionLeft
	ActionRight
	ActionQuit
)

type Reader struct {
	screen  tcell.Screen
	actions chan Action
	stop    chan struct{}
}

func New(screen tcell.Screen) *Reader {
	reader := &Reader{
		screen:  screen,
		actions: make(chan Action, 16),
		stop:    make(chan struct{}),
	}
	go reader.loop()
	return reader
}

func (reader *Reader) Close() {
	close(reader.stop)
}

func (reader *Reader) ReadAction() Action {
	select {
	case action := <-reader.actions:
		return action
	default:
		return ActionNone
	}
}

func (reader *Reader) Screen() tcell.Screen {
	return reader.screen
}

func (reader *Reader) loop() {
	for {
		select {
		case <-reader.stop:
			return
		default:
		}
		event := reader.screen.PollEvent()
		if event == nil {
			continue
		}
		action := parseEvent(event)
		if action != ActionNone {
			select {
			case reader.actions <- action:
			default:
			}
		}
	}
}

func parseEvent(event tcell.Event) Action {
	switch typed := event.(type) {
	case *tcell.EventKey:
		switch typed.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			return ActionQuit
		case tcell.KeyUp:
			return ActionUp
		case tcell.KeyDown:
			return ActionDown
		case tcell.KeyLeft:
			return ActionLeft
		case tcell.KeyRight:
			return ActionRight
		case tcell.KeyRune:
			switch typed.Rune() {
			case 'w', 'W':
				return ActionUp
			case 's', 'S':
				return ActionDown
			case 'a', 'A':
				return ActionLeft
			case 'd', 'D':
				return ActionRight
			case 'q', 'Q':
				return ActionQuit
			}
		}
	}
	return ActionNone
}
