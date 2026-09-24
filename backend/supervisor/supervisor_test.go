package supervisor

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDrainBuildProgressSuccess(t *testing.T) {
	stream := `{"stream":"Step 1/6 : FROM ocaml/opam:debian-13-ocaml-5.6\n"}` + "\n" +
		`{"stream":"Successfully tagged codingtest-harness:latest\n"}` + "\n"
	if err := drainBuildProgress(strings.NewReader(stream)); err != nil {
		t.Fatalf("drainBuildProgress: %v", err)
	}
}

func TestDrainBuildProgressReportsDaemonError(t *testing.T) {
	stream := `{"stream":"Step 3/6 : COPY utf /utf\n"}` + "\n" +
		`{"errorDetail":{"code":1,"message":"COPY failed: no source files were specified"},"error":"COPY failed: no source files were specified"}` + "\n"
	err := drainBuildProgress(strings.NewReader(stream))
	if err == nil {
		t.Fatal("drainBuildProgress returned nil for a failing build")
	}
	if !strings.Contains(err.Error(), "COPY failed: no source files were specified") {
		t.Errorf("error %q does not carry the daemon message", err)
	}
}

// newLiveSupervisor returns a supervisor that is connected to a reachable
// daemon, or skips the test. A plain `go test` run inside the backend container
// has no DOCKER_HOST until the dind service is wired up, and these tests must
// never fail for that reason.
func newLiveSupervisor(t *testing.T) *Supervisor {
	t.Helper()
	if os.Getenv("DOCKER_HOST") == "" {
		t.Skip("DOCKER_HOST is not set; skipping Docker-dependent test")
	}
	s := &Supervisor{}
	if err := s.Init(); err != nil {
		t.Skipf("docker client init failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Health(ctx); err != nil {
		t.Skipf("docker daemon unreachable: %v", err)
	}
	return s
}

func TestHealthLive(t *testing.T) {
	s := newLiveSupervisor(t)
	if err := s.Health(context.Background()); err != nil {
		t.Fatalf("Health: %v", err)
	}
}

func TestListContainersLive(t *testing.T) {
	s := newLiveSupervisor(t)
	infos, err := s.ListContainers(context.Background(), map[string]string{"codingtest.test": "absent"})
	if err != nil {
		t.Fatalf("ListContainers: %v", err)
	}
	for _, info := range infos {
		if info.Labels["codingtest.test"] != "absent" {
			t.Errorf("container %s leaked past the label filter: %v", info.ID, info.Labels)
		}
	}
}

func TestHasImageMissingLive(t *testing.T) {
	s := newLiveSupervisor(t)
	ok, err := s.HasImage(context.Background(), "codingtest-absent-image:does-not-exist")
	if err != nil {
		t.Fatalf("HasImage: %v", err)
	}
	if ok {
		t.Error("HasImage reported a non-existent image as present")
	}
}
