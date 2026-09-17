package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type DockerID = string
type SessionID = string

type Container struct {
	lock      sync.Mutex
	dockerID  DockerID
	sessionID SessionID
}

// Maybe make supervisor do containers and sessions together
type Supervisor struct {
	liveContainers   map[SessionID]*Container
	containerCounter uint16
	client           *client.Client
}

func (s *Supervisor) Init() error {
	apiClient, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	s.client = apiClient
	s.liveContainers = make(map[SessionID]*Container)
	s.containerCounter = 0

	return nil
}

func (s *Supervisor) Cleanup() error {
	for _, container := range s.liveContainers {
		container.lock.Lock()
		// TODO: Set timeout here in the stop options
		_, err := s.client.ContainerStop(context.Background(), container.dockerID, client.ContainerStopOptions{})
		container.lock.Unlock()
		if err != nil {
			// Temporary solution, maybe kill -9?
			delete(s.liveContainers, container.sessionID)

			return err
		}
	}

	return nil
}

// Kill container
func (s *Supervisor) KillContainer(id SessionID) error {
	victim, ok := s.liveContainers[id]
	if !ok {
		return fmt.Errorf("Container was not found")
	}
	victim.lock.Lock()
	defer victim.lock.Unlock()

	// TOOD: Configure kill options
	_, err := s.client.ContainerKill(context.Background(), victim.dockerID, client.ContainerKillOptions{})

	delete(s.liveContainers, id)

	return err
}

// Graceful shutdown
func (s *Supervisor) StopContainer(id SessionID) error {
	target, ok := s.liveContainers[id]
	if !ok {
		return fmt.Errorf("Container was not found")
	}
	target.lock.Lock()
	defer target.lock.Unlock()
	// TODO: Configure stop options
	_, err := s.client.ContainerStop(context.Background(), target.dockerID, client.ContainerStopOptions{})

	delete(s.liveContainers, id)

	return err
}

func (s *Supervisor) CreateContainer(image string, id SessionID) error {
	options := client.ContainerCreateOptions{
		Image: image,
		HostConfig: &container.HostConfig{
			Runtime: "runsc",
		},
	}

	createResult, err := s.client.ContainerCreate(context.Background(), options)
	if err != nil {
		return err
	}

	cont := Container{
		dockerID:  createResult.ID,
		sessionID: id,
	}

	s.liveContainers[id] = &cont

	return nil
}

func (s *Supervisor) StartContainer(id SessionID) (err error) {
	target, ok := s.liveContainers[id]
	if !ok {
		err = fmt.Errorf("container was not found")
		return
	}
	target.lock.Lock()
	defer target.lock.Unlock()

	options := client.ContainerStartOptions{}

	_, err = s.client.ContainerStart(context.Background(), target.dockerID, options)
	return
}

// executes a shell command in cwd - used by system, not user
func (s *Supervisor) Exec(id SessionID, command string) (output string, exitCode int, err error) {
	target, ok := s.liveContainers[id]
	if !ok {
		err = fmt.Errorf("container was not found")
		return
	}
	target.lock.Lock()
	defer target.lock.Unlock()

	// TODO: Configure exec create options so it actually works for the purposes of this function.
	_, err = s.client.ExecCreate(context.Background(), target.dockerID, client.ExecCreateOptions{})
	return
}

// Change working directory - used by system, not user
// func (s *Supervisor) Cwd(id SessionID, nwd string) error
// func (s *Supervisor) StreamStdout(id SessionID, to chan<- string, wg *sync.WaitGroup) error
// func (s *Supervisor) StreamStderr(id SessionID, to chan<- string, wg *sync.WaitGroup) error

// func (s *Supervisor) GetTotalMemoryUsage() (usageMb float32, err error)
func (s *Supervisor) GetContainerMemoryUsage(id SessionID) (usageMb float32, err error) {
	target, ok := s.liveContainers[id]
	if !ok {
		err = fmt.Errorf("container was not found")
		return
	}
	target.lock.Lock()
	defer target.lock.Unlock()
	resp, err := s.client.ContainerStats(context.Background(), target.dockerID, client.ContainerStatsOptions{Stream: false})
	if err != nil {
		return
	}

	defer resp.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return 0, err
	}

	return float32(stats.MemoryStats.Usage), nil
}

// TODO: Make methods for file editing so user can work on project.
