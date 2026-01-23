package render

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"pacman-pod-killer/internal/maze"
)

type Renderer struct {
	screen tcell.Screen
	style  tcell.Style
}

func New(screen tcell.Screen) *Renderer {
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	return &Renderer{screen: screen, style: style}
}

func (renderer *Renderer) Close() {
}

type Frame struct {
	Maze       maze.Grid
	PlayerX    int // Logical X coordinate
	PlayerY    int // Logical Y coordinate
	PlayerLive bool
	Enemies    []Position // Logical coordinates
	Seed       int64
	Tick       int64
	DockerInfo string
	DockerErr  string
}

type Position struct {
	X int // Logical X
	Y int // Logical Y
}

func (renderer *Renderer) Draw(frame Frame) error {
	renderer.screen.Clear()

	// Draw maze
	for y := 0; y < frame.Maze.Height; y++ {
		for x := 0; x < frame.Maze.Width; x++ {
			cell := frame.Maze.Cells[y][x]
			style := renderer.style
			if cell == maze.Wall {
				style = style.Foreground(tcell.ColorBlue)
			}
			renderer.screen.SetContent(x, y, cell, nil, style)
		}
	}

	// Draw enemies as 3x3 blocks
	enemyStyle := renderer.style.Foreground(tcell.ColorRed)
	for _, enemy := range frame.Enemies {
		sx, sy := maze.LogicToScreen(enemy.X, enemy.Y)
		renderer.drawEntity(sx, sy, 'X', enemyStyle)
	}

	// Draw player as 3x3 block
	if frame.PlayerLive {
		playerStyle := renderer.style.Foreground(tcell.ColorYellow)
		sx, sy := maze.LogicToScreen(frame.PlayerX, frame.PlayerY)
		renderer.drawEntity(sx, sy, 'C', playerStyle)
	}

	// Draw status bar below the maze
	status := fmt.Sprintf("seed:%d enemies:%d tick:%d %s", frame.Seed, len(frame.Enemies), frame.Tick, frame.DockerInfo)
	for i, r := range status {
		renderer.screen.SetContent(i, frame.Maze.Height, r, nil, renderer.style)
	}

	// Draw error line if present
	if frame.DockerErr != "" {
		errorLine := frame.Maze.Height + 1
		errorText := fmt.Sprintf("docker error: %s", frame.DockerErr)
		for i, r := range errorText {
			if i >= frame.Maze.Width {
				break
			}
			renderer.screen.SetContent(i, errorLine, r, nil, renderer.style.Foreground(tcell.ColorRed))
		}
	}

	renderer.screen.Show()
	return nil
}

// drawEntity draws a 3x3 entity at the given screen coordinates
func (renderer *Renderer) drawEntity(sx, sy int, ch rune, style tcell.Style) {
	for dy := 0; dy < maze.EntitySize; dy++ {
		for dx := 0; dx < maze.EntitySize; dx++ {
			renderer.screen.SetContent(sx+dx, sy+dy, ch, nil, style)
		}
	}
}

func (renderer *Renderer) Screen() tcell.Screen {
	return renderer.screen
}
