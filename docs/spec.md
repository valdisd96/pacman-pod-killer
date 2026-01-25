

 At the moment, I only see location1 > location2 > Win. I ran it with the command:
DOCKER_HOST=unix:///Users/uvauchok/.docker/run/docker.sock script -q /dev/null go run ./cmd/pacman --width 44 --height 12 --seed 123 --respawn-delay 10000 --epsilon 0.3 --tick-ms 80 --log-file game.log --enemy-speed 4 --locations "location1,location2,location3"
I want to display all locations, and then WIN - which means the game is completed, exiting with scores displayed (killed enemies, for example) 