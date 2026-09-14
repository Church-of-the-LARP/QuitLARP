package supervisor

import "sync"

// Maybe make supervisor do containers and sessions together
type Supervisor struct {
	liveContainers   map[SessionID]Container
	containerCounter uint16
}

// To be used as a goroutine
func (s *Supervisor) Start() error
func (s *Supervisor) Stop() error

// Kill container
func (s *Supervisor) KillContainer(id SessionID)

// Graceful shutdown
func (s *Supervisor) StopContainer(id SessionID) error
func (s *Supervisor) StartContainer(image string, sessionID SessionID) (err error)

// executes a shell command in cwd - used by system, not user
func (s *Supervisor) Exec(id SessionID, command string) (output string, exitCode int, err error)

// Change working directory - used by system, not user
func (s *Supervisor) Cwd(id SessionID, nwd string) error
func (s *Supervisor) StreamStdout(id SessionID, to chan<- string, wg *sync.WaitGroup) error
func (s *Supervisor) StreamStderr(id SessionID, to chan<- string, wg *sync.WaitGroup) error

func (s *Supervisor) GetTotalMemoryUsage() (usageMb float32, err error)
func (s *Supervisor) GetContainerMemoryUsage(id SessionID) (usageMb float32, err error)

// TODO: Make methods for file editing so user can work on project.

type GVisorID = string
type ProcID = int
type DockerID = string
type SessionID = string
type TaskID = string

type Container struct {
	gVisorID  GVisorID
	procID    ProcID
	dockerID  DockerID
	sessionID SessionID
	taskID    TaskID
}
