// Package supervisor is the Docker client layer for the sandboxed assessment
// runtime. It talks to the dind daemon (DOCKER_HOST, plain TCP on the compose
// network) and launches every untrusted submission container with the runsc
// (gVisor) OCI runtime, so assessment code never runs on the host kernel.
//
// The daemon itself lives in its own container: this package only ever holds a
// moby client plus the set of sessions it created, keyed by SessionID. Timers,
// resource limits and image tags are configuration, not policy, and come from
// config.AssessmentEnv.
package supervisor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// DockerID is the daemon-assigned container id.
type DockerID = string

// SessionID is the caller-facing key for one assessment session.
type SessionID = string

// ContainerSpec describes one sandboxed container. Runtime is the OCI runtime
// name ("runsc" for gVisor). Cmd is the container's init process; when it exits
// the session is over, so callers usually pass a long lived command and drive
// everything else through Exec. MemoryBytes is a byte count, NanoCPUs is in
// units of 1e-9 CPU and PidsLimit is a process count; each zero value leaves
// that limit unset. Labels are used to find the container again after a restart.
type ContainerSpec struct {
	Runtime         string // OCI runtime, "runsc"
	Cmd             []string
	WorkDir         string
	Env             []string // KEY=value
	NetworkDisabled bool
	MemoryBytes     int64
	NanoCPUs        int64
	PidsLimit       int64
	Labels          map[string]string
}

// ExecSpec describes one command run inside an existing container.
type ExecSpec struct {
	Cmd     []string
	WorkDir string
	Env     []string
	Timeout time.Duration // bounds the whole exec; zero means no timeout
	Stdin   []byte        // written to the command's stdin, which is then closed
}

// ExecResult is the combined stdout+stderr of an exec and its exit code.
// ExitCode is -1 when the command could not be run to completion (timeout or
// transport error); the error returned alongside it explains why.
type ExecResult struct {
	Output   string
	ExitCode int
}

// ContainerInfo is the subset of a listed container the assessment layer needs.
type ContainerInfo struct {
	ID     string
	Image  string
	State  string
	Labels map[string]string
}

// sessionContainer tracks one live container behind a mutex so Kill/Remove and
// Exec cannot interleave on the same session.
type sessionContainer struct {
	lock      sync.Mutex
	dockerID  DockerID
	sessionID SessionID
}

// Supervisor owns the docker client and the containers it created.
type Supervisor struct {
	liveContainers map[SessionID]*sessionContainer
	client         *client.Client
}

// startWaitTimeout bounds how long StartContainer polls for the running state.
const startWaitTimeout = 15 * time.Second

// fileExecTimeout bounds one file read or write through the sandbox.
const fileExecTimeout = 30 * time.Second

// Init builds the docker client from the environment (DOCKER_HOST) and resets
// the session table. It does not contact the daemon: call Health for that.
func (s *Supervisor) Init() error {
	apiClient, err := client.New(client.FromEnv)
	if err != nil {
		return fmt.Errorf("supervisor: create docker client: %w", err)
	}

	s.client = apiClient
	if s.liveContainers == nil {
		s.liveContainers = make(map[SessionID]*sessionContainer)
	}

	return nil
}

// Cleanup force-removes every container this supervisor created. It keeps
// going after individual failures and reports them together, so one stuck
// container cannot block the rest of the shutdown.
func (s *Supervisor) Cleanup() error {
	ctx := context.Background()
	var errs []error
	for id, target := range s.liveContainers {
		target.lock.Lock()
		_, killErr := s.client.ContainerKill(ctx, target.dockerID, client.ContainerKillOptions{Signal: "SIGKILL"})
		if killErr != nil && !cerrdefs.IsNotFound(killErr) {
			errs = append(errs, fmt.Errorf("supervisor: kill %s: %w", id, killErr))
		}
		_, removeErr := s.client.ContainerRemove(ctx, target.dockerID, client.ContainerRemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		})
		target.lock.Unlock()
		if removeErr != nil && !cerrdefs.IsNotFound(removeErr) {
			errs = append(errs, fmt.Errorf("supervisor: remove %s: %w", id, removeErr))
		}
		delete(s.liveContainers, id)
	}
	return errors.Join(errs...)
}

// Health pings the daemon. It is the readiness probe for the dind dependency.
func (s *Supervisor) Health(ctx context.Context) error {
	if s.client == nil {
		return errors.New("supervisor: not initialized")
	}
	if _, err := s.client.Ping(ctx, client.PingOptions{}); err != nil {
		return fmt.Errorf("supervisor: docker daemon unreachable: %w", err)
	}
	return nil
}

// CreateContainer creates (but does not start) a container for one session.
// The container has no TTY and no attached streams: every command runs through
// Exec, which needs a multiplexed (non-TTY) stream to split stdout and stderr.
func (s *Supervisor) CreateContainer(image string, id SessionID, spec ContainerSpec) error {
	cfg := &container.Config{
		Image:      image,
		Cmd:        spec.Cmd,
		WorkingDir: spec.WorkDir,
		Env:        spec.Env,
		Labels:     spec.Labels,
	}

	hostCfg := &container.HostConfig{
		Runtime: spec.Runtime,
		Resources: container.Resources{
			Memory:   spec.MemoryBytes,
			NanoCPUs: spec.NanoCPUs,
		},
	}
	// PidsLimit is a pointer: nil leaves the daemon default, a value caps it.
	if spec.PidsLimit > 0 {
		limit := spec.PidsLimit
		hostCfg.PidsLimit = &limit
	}
	if spec.NetworkDisabled {
		hostCfg.NetworkMode = network.NetworkNone
	}

	result, err := s.client.ContainerCreate(context.Background(), client.ContainerCreateOptions{
		Config:     cfg,
		HostConfig: hostCfg,
	})
	if err != nil {
		return fmt.Errorf("supervisor: create container for %s: %w", id, err)
	}

	s.liveContainers[id] = &sessionContainer{dockerID: result.ID, sessionID: id}
	return nil
}

// StartContainer starts the session container and waits until the daemon
// reports it running, so a following Exec does not race the start.
func (s *Supervisor) StartContainer(id SessionID) error {
	target, ok := s.liveContainers[id]
	if !ok {
		return fmt.Errorf("supervisor: session %s not found", id)
	}
	target.lock.Lock()
	defer target.lock.Unlock()

	ctx := context.Background()
	if _, err := s.client.ContainerStart(ctx, target.dockerID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("supervisor: start container for %s: %w", id, err)
	}
	return s.waitRunning(ctx, target.dockerID)
}

// waitRunning polls the container until it is running. Start returns as soon as
// the daemon accepted the request, which under gVisor can be well before the
// init process is actually up.
func (s *Supervisor) waitRunning(ctx context.Context, dockerID string) error {
	deadline := time.Now().Add(startWaitTimeout)
	for {
		res, err := s.client.ContainerInspect(ctx, dockerID, client.ContainerInspectOptions{})
		if err != nil {
			return fmt.Errorf("supervisor: inspect %s: %w", dockerID, err)
		}
		state := res.Container.State
		if state != nil {
			if state.Running {
				return nil
			}
			// A container that already reached a terminal state will never
			// become running; report it now instead of polling to the timeout.
			if state.Status == container.StateExited || state.Dead {
				return fmt.Errorf("supervisor: container %s exited before it was running (exit code %d)", dockerID, state.ExitCode)
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("supervisor: container %s did not reach the running state within %s", dockerID, startWaitTimeout)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// KillContainer sends SIGKILL to the session container and forgets the session.
func (s *Supervisor) KillContainer(id SessionID) error {
	target, ok := s.liveContainers[id]
	if !ok {
		return fmt.Errorf("supervisor: session %s not found", id)
	}
	target.lock.Lock()
	defer target.lock.Unlock()

	_, err := s.client.ContainerKill(context.Background(), target.dockerID, client.ContainerKillOptions{Signal: "SIGKILL"})
	if err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("supervisor: kill container for %s: %w", id, err)
	}

	delete(s.liveContainers, id)
	return nil
}

// RemoveContainer force-removes the session container and its volumes.
func (s *Supervisor) RemoveContainer(id SessionID) error {
	target, ok := s.liveContainers[id]
	if !ok {
		return fmt.Errorf("supervisor: session %s not found", id)
	}
	target.lock.Lock()
	defer target.lock.Unlock()

	_, err := s.client.ContainerRemove(context.Background(), target.dockerID, client.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("supervisor: remove container for %s: %w", id, err)
	}

	delete(s.liveContainers, id)
	return nil
}

// RemoveDockerContainer force-removes a container by its daemon id, whether or
// not this supervisor created it. It is the boot-time orphan path: containers
// left behind by a previous backend process are found by label with
// ListContainers, so they have no session entry here.
func (s *Supervisor) RemoveDockerContainer(ctx context.Context, dockerID DockerID) error {
	if _, err := s.client.ContainerRemove(ctx, dockerID, client.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}); err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("supervisor: remove container %s: %w", dockerID, err)
	}

	// Drop any session bookkeeping that pointed at this container, so a later
	// Kill/Remove does not chase an id the daemon already forgot.
	for sessionID, target := range s.liveContainers {
		if target.dockerID == dockerID {
			delete(s.liveContainers, sessionID)
		}
	}
	return nil
}

// Exec runs one command inside the session container and returns its combined
// output and exit code. Timeout bounds the whole operation: on expiry the
// hijacked connection is closed and whatever was read so far is returned with
// a timeout error.
func (s *Supervisor) Exec(id SessionID, spec ExecSpec) (ExecResult, error) {
	target, ok := s.liveContainers[id]
	if !ok {
		return ExecResult{}, fmt.Errorf("supervisor: session %s not found", id)
	}
	target.lock.Lock()
	defer target.lock.Unlock()

	ctx := context.Background()
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}

	create, err := s.client.ExecCreate(ctx, target.dockerID, client.ExecCreateOptions{
		Cmd:          spec.Cmd,
		WorkingDir:   spec.WorkDir,
		Env:          spec.Env,
		AttachStdin:  len(spec.Stdin) > 0,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return ExecResult{}, fmt.Errorf("supervisor: create exec on %s: %w", id, err)
	}

	attach, err := s.client.ExecAttach(ctx, create.ID, client.ExecAttachOptions{})
	if err != nil {
		return ExecResult{}, fmt.Errorf("supervisor: attach exec on %s: %w", id, err)
	}
	defer attach.Close()

	if len(spec.Stdin) > 0 {
		if _, err := attach.Conn.Write(spec.Stdin); err != nil {
			return ExecResult{}, fmt.Errorf("supervisor: write stdin on %s: %w", id, err)
		}
		// Half-close so the command sees EOF while its output stays readable.
		if closer, ok := attach.Conn.(interface{ CloseWrite() error }); ok {
			if err := closer.CloseWrite(); err != nil {
				return ExecResult{}, fmt.Errorf("supervisor: close stdin on %s: %w", id, err)
			}
		} else {
			attach.Close()
		}
	}

	// The exec has no TTY, so stdout and stderr arrive as multiplexed frames
	// with an 8 byte header each. Demultiplex into one buffer so the caller
	// sees the two streams in the order they were written.
	var out bytes.Buffer
	done := make(chan error, 1)
	go func() {
		_, copyErr := stdcopy.StdCopy(&out, &out, attach.Reader)
		done <- copyErr
	}()

	select {
	case copyErr := <-done:
		if copyErr != nil && !errors.Is(copyErr, io.EOF) {
			return ExecResult{Output: out.String(), ExitCode: -1}, fmt.Errorf("supervisor: read exec stream on %s: %w", id, copyErr)
		}
	case <-ctx.Done():
		// Closing the hijacked connection unblocks the reader goroutine; wait
		// for it before touching the buffer so the read is race free.
		attach.Close()
		<-done
		return ExecResult{Output: out.String(), ExitCode: -1}, fmt.Errorf("supervisor: exec on %s timed out after %s", id, spec.Timeout)
	}

	inspect, err := s.client.ExecInspect(context.Background(), create.ID, client.ExecInspectOptions{})
	if err != nil {
		return ExecResult{Output: out.String(), ExitCode: -1}, fmt.Errorf("supervisor: inspect exec on %s: %w", id, err)
	}
	return ExecResult{Output: out.String(), ExitCode: inspect.ExitCode}, nil
}

// ReadFile returns the contents of a file inside the container. Both this and
// WriteFile go through a command in the sandbox rather than the daemon's
// archive endpoints: the sandbox's file view is the one the assessment code
// reads and writes, and the daemon's view of a running gVisor sandbox can lag
// behind it.
func (s *Supervisor) ReadFile(id SessionID, path string) ([]byte, error) {
	result, err := s.Exec(id, ExecSpec{
		Cmd:     []string{"sh", "-c", "cat -- " + shellQuote(path)},
		Timeout: fileExecTimeout,
	})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("supervisor: read %s from %s: cat exited %d: %s",
			path, id, result.ExitCode, strings.TrimSpace(result.Output))
	}
	return []byte(result.Output), nil
}

// WriteFile replaces the contents of a file inside the container; its
// directory must already exist.
func (s *Supervisor) WriteFile(id SessionID, path string, content []byte) error {
	result, err := s.Exec(id, ExecSpec{
		Cmd:     []string{"sh", "-c", "cat > " + shellQuote(path)},
		Stdin:   content,
		Timeout: fileExecTimeout,
	})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("supervisor: write %s in %s: command exited %d: %s",
			path, id, result.ExitCode, strings.TrimSpace(result.Output))
	}
	return nil
}

// shellQuote single-quotes a path for sh, escaping embedded single quotes.
func shellQuote(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}

// CopyFromDir copies a directory from the container onto the host, unpacking
// it into destDir, which is created when missing. The archive members are
// rooted at the source directory's own name, which is stripped so destDir
// holds its contents. The extractor refuses members that would escape destDir.
func (s *Supervisor) CopyFromDir(id SessionID, sourcePath, destDir string) error {
	target, ok := s.liveContainers[id]
	if !ok {
		return fmt.Errorf("supervisor: session %s not found", id)
	}
	target.lock.Lock()
	defer target.lock.Unlock()

	res, err := s.client.CopyFromContainer(context.Background(), target.dockerID, client.CopyFromContainerOptions{SourcePath: sourcePath})
	if err != nil {
		return fmt.Errorf("supervisor: copy %s out of %s: %w", sourcePath, id, err)
	}
	defer res.Content.Close()

	if err := extractTar(res.Content, destDir, path.Base(sourcePath)); err != nil {
		return fmt.Errorf("supervisor: unpack %s from %s: %w", sourcePath, id, err)
	}
	return nil
}

// BuildImage builds an image from a tar build context and tags it. The daemon's
// JSON progress stream is drained and any error it reports is surfaced so the
// caller can show the Dockerfile failure instead of a bare HTTP status.
func (s *Supervisor) BuildImage(ctx context.Context, tag string, contextTar io.Reader, dockerfilePath string) error {
	res, err := s.client.ImageBuild(ctx, contextTar, client.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: dockerfilePath,
		Remove:     true,
	})
	if err != nil {
		return fmt.Errorf("supervisor: build %s: %w", tag, err)
	}
	defer res.Body.Close()

	if err := drainBuildProgress(res.Body); err != nil {
		return fmt.Errorf("supervisor: build %s: %w", tag, err)
	}
	return nil
}

// HasImage reports whether ref exists on the daemon. A missing image is not an
// error.
func (s *Supervisor) HasImage(ctx context.Context, ref string) (bool, error) {
	if _, err := s.client.ImageInspect(ctx, ref); err != nil {
		if cerrdefs.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("supervisor: inspect image %s: %w", ref, err)
	}
	return true, nil
}

// ListContainers returns every container (running or not) that carries all of
// the given labels.
func (s *Supervisor) ListContainers(ctx context.Context, labelFilter map[string]string) ([]ContainerInfo, error) {
	filters := make(client.Filters, len(labelFilter))
	for key, value := range labelFilter {
		filters = filters.Add("label", key+"="+value)
	}

	res, err := s.client.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, fmt.Errorf("supervisor: list containers: %w", err)
	}

	infos := make([]ContainerInfo, 0, len(res.Items))
	for _, item := range res.Items {
		infos = append(infos, ContainerInfo{
			ID:     item.ID,
			Image:  item.Image,
			State:  string(item.State),
			Labels: item.Labels,
		})
	}
	return infos, nil
}

// drainBuildProgress reads the daemon's JSON progress lines and returns an
// error carrying the daemon's message when the build failed. Unknown fields
// are ignored so the same decoder works for the classic and BuildKit streams.
// The tail of the plain output is kept so a failing RUN command shows what the
// command itself reported.
func drainBuildProgress(r io.Reader) error {
	const tailLimit = 4000
	dec := json.NewDecoder(r)
	var failures []string
	var tail []byte
	for {
		var msg struct {
			Error       string `json:"error"`
			ErrorDetail *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"errorDetail"`
			Stream string `json:"stream"`
		}
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read build progress: %w", err)
		}
		if msg.Stream != "" {
			tail = append(tail, msg.Stream...)
			if len(tail) > tailLimit {
				tail = tail[len(tail)-tailLimit:]
			}
		}
		if msg.ErrorDetail != nil && msg.ErrorDetail.Message != "" {
			failures = append(failures, msg.ErrorDetail.Message)
			continue
		}
		if msg.Error != "" {
			failures = append(failures, msg.Error)
		}
	}
	if len(failures) > 0 {
		message := strings.Join(failures, "; ")
		if len(tail) > 0 {
			message += "\nlast build output:\n" + strings.TrimSpace(string(tail))
		}
		return fmt.Errorf("image build failed: %s", message)
	}
	return nil
}
