# Pacman Pod Killer

Terminal Pacman game that maps running Docker containers to enemies. When you kill an enemy, its container is removed.

## Run

macOS Docker Desktop socket example:

```bash
DOCKER_HOST=unix:///Users/uvauchok/.docker/run/docker.sock \
script -q /dev/null go run ./cmd/pacman --width 80 --height 30 --seed 123 --respawn-delay 10000 --tick-ms 80 --debug
```

General run:

```bash
go run ./cmd/pacman
```

## Controls
- Arrows or WASD to move
- Q to quit

## Warning
This game deletes running containers when enemies die.
