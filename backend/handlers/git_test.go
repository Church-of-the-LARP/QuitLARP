package handlers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"backend/config"
)

func TestGitServiceHeader(t *testing.T) {
	got := string(gitServiceHeader(gitUploadPack))
	want := "001e# service=git-upload-pack\n0000"
	if got != want {
		t.Fatalf("gitServiceHeader(%s) = %q, want %q", gitUploadPack, got, want)
	}
}

func TestGitRepoDirRejectsUnsafeNames(t *testing.T) {
	h := &Handlers{cfg: &config.Config{GitReposDir: t.TempDir()}}
	for _, name := range []string{"", ".", "..", "../evil", "a/b", ".hidden", "foo bar", strings.Repeat("x", 65)} {
		if _, err := h.gitRepoDir(name); err == nil {
			t.Errorf("gitRepoDir(%q) was accepted, want an error", name)
		}
	}
}

func TestGitRepoDirInitializesBareRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	reposDir := t.TempDir()
	h := &Handlers{cfg: &config.Config{GitReposDir: reposDir}}

	dir, err := h.gitRepoDir("demo.git")
	if err != nil {
		t.Fatalf("gitRepoDir: %v", err)
	}
	if want := filepath.Join(reposDir, "demo.git"); dir != want {
		t.Fatalf("gitRepoDir = %q, want %q", dir, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		t.Fatalf("no bare repository at %s: %v", dir, err)
	}

	// Both the ".git" and the bare id resolve to the same repo, and a second
	// call is a no-op instead of a re-init.
	if again, err := h.gitRepoDir("demo"); err != nil || again != dir {
		t.Fatalf("gitRepoDir(demo) = %q, %v; want %q, nil", again, err, dir)
	}
}
