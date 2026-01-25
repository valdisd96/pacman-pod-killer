package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
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
	defaultEpsilon       = 0.1 // SARSA exploration rate
	defaultEnemySpeed    = 2   // Enemy moves every N ticks (higher = slower enemies)
	defaultLocations     = ""  // No locations = single maze mode
)

func main() {
	width := flag.Int("width", defaultWidth, "logical maze width (corridor cells)")
	height := flag.Int("height", defaultHeight, "logical maze height (corridor cells)")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed")
	tickMillis := flag.Int("tick-ms", defaultTickMillis, "tick duration in ms")
	respawnMillis := flag.Int("respawn-delay", defaultRespawnMillis, "respawn delay in ms")
	enemySpeed := flag.Int("enemy-speed", defaultEnemySpeed, "enemy moves every N ticks (higher = slower enemies, min 1)")
	debug := flag.Bool("debug", false, "enable debug logging")
	logFile := flag.String("log-file", "", "path to log file for game actions (empty = disabled)")

	// SARSA AI flags
	aiMode := flag.String("ai-mode", "sarsa", "AI mode: 'random' or 'sarsa'")
	training := flag.Bool("training", true, "enable SARSA training mode (updates Q-table)")
	epsilon := flag.Float64("epsilon", defaultEpsilon, "SARSA exploration rate 0.0-1.0")
	qtablePath := flag.String("qtable", ai.DefaultQTablePath(), "path to Q-table file")

	// Location flags for level-based gameplay
	locations := flag.String("locations", defaultLocations, "comma-separated list of location patterns (e.g., 'location1,location2,location3')")

	flag.Parse()

	// Parse locations into a slice
	var locationList []string
	if *locations != "" {
		for _, loc := range strings.Split(*locations, ",") {
			loc = strings.TrimSpace(loc)
			if loc != "" {
				locationList = append(locationList, loc)
			}
		}
	}

	log, err := logger.New(*logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	if log.Enabled() {
		log.Info("=== Game starting ===")
		log.Info("Config: width=%d height=%d seed=%d tick=%dms respawn=%dms enemy-speed=%d", *width, *height, *seed, *tickMillis, *respawnMillis, *enemySpeed)
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "This game requires an interactive TTY. Run it in a real terminal.")
		os.Exit(1)
	}

	fmt.Println("Warning: this game deletes running containers on enemy death.")

	// Ensure enemy speed is at least 1
	espeed := *enemySpeed
	if espeed < 1 {
		espeed = 1
	}

	// Create RNG for enemy spawning
	rng := rand.New(rand.NewSource(*seed))

	var state *game.GameState
	if len(locationList) > 0 {
		// Level-based mode with locations
		state = game.NewStateWithConfig(game.StateConfig{
			Seed:         *seed,
			RespawnDelay: time.Duration(*respawnMillis) * time.Millisecond,
			EnemySpeed:   espeed,
			Locations:    locationList,
			LogicWidth:   *width,
			LogicHeight:  *height,
		}, nil)
		log.Info("Level mode: %d locations configured: %v", len(locationList), locationList)
	} else {
		// Classic single maze mode
		mazeGrid := maze.Generate(*width, *height, rng)
		state = game.NewState(mazeGrid, *seed, time.Duration(*respawnMillis)*time.Millisecond, espeed)
	}

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
				snapshots = append(snapshots, game.EnemySnapshot{
					ID:            enemy.ID,
					ContainerID:   enemy.ContainerID,
					ContainerName: enemy.ContainerName,
					Alive:         enemy.Alive,
				})
			}
			state.SyncEnemies(game.EnemiesFromDocker(snapshots, rng, state))
		}
	}

	// Initialize AI
	var sarsaAI *ai.SARSA
	var selectedAIMode game.AIMode

	if *aiMode == "sarsa" {
		selectedAIMode = game.AIModeSARSA
		// Load existing Q-table or create new one
		qTable, err := ai.LoadQTable(*qtablePath)
		if err != nil {
			log.Error("Failed to load Q-table from %s: %v (starting fresh)", *qtablePath, err)
			qTable = ai.NewQTable()
		} else if qTable.Size() > 0 {
			log.Info("Loaded Q-table with %d states from %s", qTable.Size(), *qtablePath)
		}

		sarsaAI = ai.NewSARSA(ai.SARSAConfig{
			Alpha:    ai.DefaultAlpha,
			Gamma:    ai.DefaultGamma,
			Epsilon:  *epsilon,
			Training: *training,
			QTable:   qTable,
		})
		log.Info("SARSA AI initialized: training=%v epsilon=%.2f", *training, *epsilon)
	} else {
		selectedAIMode = game.AIModeRandom
		log.Info("Random AI initialized")
	}

	controller := game.NewController(game.ControllerConfig{
		State:       state,
		Renderer:    renderer,
		Input:       inputReader,
		RandomAI:    ai.NewRandom(),
		SarsaAI:     sarsaAI,
		AIMode:      selectedAIMode,
		EnemyEvents: enemyChan,
		Docker:      dockerClient,
		Log:         log,
	})

	ticker := time.NewTicker(time.Duration(*tickMillis) * time.Millisecond)
	defer ticker.Stop()

	// Save Q-table on exit
	defer func() {
		if sarsaAI != nil && *training {
			if err := controller.SaveQTable(*qtablePath); err != nil {
				log.Error("Failed to save Q-table: %v", err)
			} else {
				log.Info("Saved Q-table to %s (%d states)", *qtablePath, sarsaAI.QTable().Size())
			}
		}
	}()

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
