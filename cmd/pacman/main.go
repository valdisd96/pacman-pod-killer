package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"golang.org/x/term"
	"pacman-pod-killer/internal/ai"
	"pacman-pod-killer/internal/dockerwatch"
	"pacman-pod-killer/internal/game"
	"pacman-pod-killer/internal/input"
	"pacman-pod-killer/internal/logger"
	"pacman-pod-killer/internal/maze"
	"pacman-pod-killer/internal/render"
)

const (
	defaultWidth         = 15 // Logical width (number of corridor cells)
	defaultHeight        = 10 // Logical height (number of corridor cells)
	defaultTickMillis    = 80
	defaultRespawnMillis = 10000
)

func main() {
	width := flag.Int("width", defaultWidth, "logical maze width (corridor cells)")
	height := flag.Int("height", defaultHeight, "logical maze height (corridor cells)")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed")
	tickMillis := flag.Int("tick-ms", defaultTickMillis, "tick duration in ms")
	respawnMillis := flag.Int("respawn-delay", defaultRespawnMillis, "respawn delay in ms")
	debug := flag.Bool("debug", false, "enable debug logging")
	logFile := flag.String("log-file", "", "path to log file for game actions (empty = disabled)")
	flag.Parse()

	log, err := logger.New(*logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	if log.Enabled() {
		log.Info("=== Game starting ===")
		log.Info("Config: width=%d height=%d seed=%d tick=%dms respawn=%dms", *width, *height, *seed, *tickMillis, *respawnMillis)
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "This game requires an interactive TTY. Run it in a real terminal.")
		os.Exit(1)
	}

	fmt.Println("Warning: this game deletes running containers on enemy death.")

	rng := rand.New(rand.NewSource(*seed))
	mazeGrid := maze.Generate(*width, *height, rng)

	state := game.NewState(mazeGrid, *seed, time.Duration(*respawnMillis)*time.Millisecond)

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "screen init failed: %v\n", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "screen setup failed: %v\n", err)
		os.Exit(1)
	}
	screen.Clear()
	defer screen.Fini()

	renderer := render.New(screen)
	defer renderer.Close()

	inputReader := input.New(screen)
	defer inputReader.Close()

	var dockerClient *dockerwatch.Client
	var enemyChan chan dockerwatch.Event
	var stopChan chan struct{}

	client, err := dockerwatch.New(log)
	if err != nil {
		state.DockerStatus = "docker: disconnected"
		state.DockerError = err.Error()
		log.Error("Docker init failed: %v", err)
		fmt.Fprintf(os.Stderr, "docker init failed (running without docker): %v\n", err)
	} else {
		dockerClient = client
		dockerClient.SetDebug(*debug)
		enemyChan = make(chan dockerwatch.Event, 32)
		stopChan = make(chan struct{})
		go dockerClient.Watch(enemyChan, stopChan)

		enemies, err := dockerClient.ListEnemies()
		if err != nil {
			state.DockerStatus = "docker: unavailable"
			state.DockerError = err.Error()
			log.Error("Docker list failed: %v", err)
			fmt.Fprintf(os.Stderr, "docker list failed (starting with no enemies): %v\n", err)
		} else {
			state.DockerStatus = "docker: connected"
			state.DockerError = ""
			log.Info("Docker connected, found %d containers", len(enemies))
			snapshots := make([]game.EnemySnapshot, 0, len(enemies))
			for _, enemy := range enemies {
				snapshots = append(snapshots, game.EnemySnapshot{ID: enemy.ID, ContainerID: enemy.ContainerID, Alive: enemy.Alive})
			}
			state.SyncEnemies(game.EnemiesFromDocker(snapshots, rng, state))
		}
	}

	controller := game.NewController(state, renderer, inputReader, ai.NewRandom(), enemyChan, dockerClient, log)
	ticker := time.NewTicker(time.Duration(*tickMillis) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := controller.Tick(); err != nil {
				if stopChan != nil {
					close(stopChan)
				}
				if err == game.ErrQuit {
					return
				}
				fmt.Fprintf(os.Stderr, "game loop error: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
