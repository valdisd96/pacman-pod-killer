package dockerwatch

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"pacman-pod-killer/internal/logger"
)

type EventType string

const (
	EventStart EventType = "start"
	EventStop  EventType = "stop"
)

type Event struct {
	Type          EventType
	ContainerID   string
	ContainerName string // Container name for location matching
}

type Client struct {
	docker *client.Client
	debug  bool
	log    *logger.Logger
}

type EnemyInfo struct {
	ID            string
	ContainerID   string
	ContainerName string // Container name for location matching
	Alive         bool
}

func New(log *logger.Logger) (*Client, error) {
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{docker: docker, log: log}, nil
}

func (c *Client) SetDebug(enabled bool) {
	c.debug = enabled
}

func (c *Client) ListEnemies() ([]EnemyInfo, error) {
	c.log.Debug("Listing containers...")
	containers, err := c.docker.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		c.log.Error("ContainerList failed: %v", err)
		return nil, err
	}
	if c.debug {
		fmt.Printf("dockerwatch: listed %d containers\n", len(containers))
	}
	c.log.Info("Listed %d containers", len(containers))
	enemies := make([]EnemyInfo, 0, len(containers))
	for _, ctr := range containers {
		id := ctr.ID
		if len(id) > 12 {
			id = id[:12]
		}
		// Extract container name (remove leading /)
		name := ""
		if len(ctr.Names) > 0 {
			name = ctr.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}
		c.log.Debug("Found container: id=%s name=%s full_id=%s", id, name, ctr.ID)
		enemies = append(enemies, EnemyInfo{ID: id, ContainerID: ctr.ID, ContainerName: name, Alive: true})
	}
	return enemies, nil
}

func (c *Client) Watch(eventsOut chan<- Event, stop <-chan struct{}) {
	c.log.Info("Starting Docker event watcher")
	filter := filters.NewArgs()
	filter.Add("type", "container")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages, errors := c.docker.Events(ctx, types.EventsOptions{Filters: filter})

	for {
		select {
		case <-stop:
			c.log.Info("Docker watcher stopped")
			return
		case err := <-errors:
			if err != nil {
				c.log.Error("Docker events error: %v", err)
				time.Sleep(250 * time.Millisecond)
			}
		case message := <-messages:
			if message.Type != events.ContainerEventType {
				continue
			}
			// Extract container name from event attributes
			name := ""
			if message.Actor.Attributes != nil {
				name = message.Actor.Attributes["name"]
			}
			c.log.Debug("Docker event: action=%s container=%s name=%s", message.Action, message.ID, name)
			if message.Action == "start" {
				c.log.Info("Container started: %s (name=%s)", message.ID, name)
				eventsOut <- Event{Type: EventStart, ContainerID: message.ID, ContainerName: name}
			}
			if message.Action == "die" || message.Action == "stop" {
				c.log.Info("Container stopped: %s (name=%s, action=%s)", message.ID, name, message.Action)
				eventsOut <- Event{Type: EventStop, ContainerID: message.ID, ContainerName: name}
			}
		}
	}
}

func (c *Client) Remove(containerID string) error {
	if containerID == "" {
		c.log.Error("Remove called with empty container ID")
		return fmt.Errorf("empty container id")
	}
	c.log.Info("Removing container: %s", containerID)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := c.docker.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
	if err != nil {
		c.log.Error("ContainerRemove failed for %s: %v", containerID, err)
	} else {
		c.log.Info("Container removed successfully: %s", containerID)
	}
	return err
}
