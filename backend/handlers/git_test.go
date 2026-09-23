package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
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
	for _, name := range []string{
		"", ".", "..", "../evil", "evil/..", "a/../b", "/a", "a/", "a//b",
		".hidden", "a/.hidden", "foo bar", strings.Repeat("x", 65), strings.Repeat("a/", 130),
	} {
		if _, err := h.gitRepoDir(name); err == nil {
			t.Errorf("gitRepoDir(%q) was accepted, want an error", name)
		}
	}
}

func TestGitRepoDirInitializesNestedRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	reposDir := t.TempDir()
	h := &Handlers{cfg: &config.Config{GitReposDir: reposDir}}

	dir, err := h.gitRepoDir("assessments/assessment-1.git")
	if err != nil {
		t.Fatalf("gitRepoDir: %v", err)
	}
	if want := filepath.Join(reposDir, "assessments", "assessment-1.git"); dir != want {
		t.Fatalf("gitRepoDir = %q, want %q", dir, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		t.Fatalf("no bare repository at %s: %v", dir, err)
	}

	// The ".git" suffix is optional, and a second call is a no-op.
	if again, err := h.gitRepoDir("assessments/assessment-1"); err != nil || again != dir {
		t.Fatalf("gitRepoDir(assessments/assessment-1) = %q, %v; want %q, nil", again, err, dir)
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

func TestParseAssessmentRepoID(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"assessments/assessment-1", 1, true},
		{"assessments/assessment-1.git", 1, true},
		{"assessments/assessment-42", 42, true},
		{"assessments/assessment-0", 0, false},
		{"assessments/assessment-01", 0, false},
		{"assessments/assessment-007", 0, false},
		{"assessments/assessment-", 0, false},
		{"assessments/assessment-abc", 0, false},
		{"assessments/assessment-1x", 0, false},
		{"assessments/assessment-1/extra", 0, false},
		{"assessments/assessment-1/2", 0, false},
		{"assessments/demo", 0, false},
		{"assessment-1", 0, false},
		{"demo", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, ok := parseAssessmentRepoID(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseAssessmentRepoID(%q) = %d, %v; want %d, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestGitRepoDirInstallsPushHookForAssessments(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	reposDir := t.TempDir()
	h := &Handlers{cfg: &config.Config{GitReposDir: reposDir}}

	dir, err := h.gitRepoDir("assessments/assessment-1")
	if err != nil {
		t.Fatalf("gitRepoDir: %v", err)
	}
	hook := filepath.Join(dir, "hooks", "update")
	info, err := os.Stat(hook)
	if err != nil {
		t.Fatalf("no push hook installed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Errorf("hook mode = %#o, want 0755", perm)
	}
	first, err := os.ReadFile(hook)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !strings.Contains(string(first), "PUSH_VALIDATION_URL") {
		t.Errorf("hook does not mention PUSH_VALIDATION_URL:\n%s", first)
	}

	// A second call leaves the hook untouched.
	if _, err := h.gitRepoDir("assessments/assessment-1.git"); err != nil {
		t.Fatalf("second gitRepoDir: %v", err)
	}
	second, err := os.ReadFile(hook)
	if err != nil {
		t.Fatalf("read hook again: %v", err)
	}
	if string(second) != string(first) {
		t.Errorf("hook changed on a second call:\n%s", second)
	}

	// Repositories that do not back an assessment get no hook.
	demoDir, err := h.gitRepoDir("demo")
	if err != nil {
		t.Fatalf("gitRepoDir(demo): %v", err)
	}
	if _, err := os.Stat(filepath.Join(demoDir, "hooks", "update")); !os.IsNotExist(err) {
		t.Errorf("non-assessment repo has a push hook (stat err = %v)", err)
	}
}

func TestPushHookBehaves(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	hook := filepath.Join(t.TempDir(), "update")
	if err := os.WriteFile(hook, []byte(pushHook), 0o755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	// One stub curl per behaviour, each in its own bin directory.
	stub := func(name, script string) string {
		t.Helper()
		bin := t.TempDir()
		if err := os.WriteFile(filepath.Join(bin, "curl"), []byte(script), 0o755); err != nil {
			t.Fatalf("write %s stub: %v", name, err)
		}
		return bin
	}
	acceptBin := stub("accept", "#!/bin/sh\ncat >/dev/null\nprintf 'HTTP_STATUS:200'\nexit 0\n")
	rejectBin := stub("reject", "#!/bin/sh\ncat >/dev/null\nprintf 'push to main was rejected:\\n- chapter 1: broken\\nHTTP_STATUS:422'\nexit 0\n")
	downBin := stub("down", "#!/bin/sh\ncat >/dev/null\nexit 7\n")
	neverBin := stub("never", "#!/bin/sh\nexit 99\n")

	// Start from a clean environment so the test cannot inherit a secret.
	base := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PUSH_VALIDATION_") || strings.HasPrefix(e, "PUSH_REPOSITORY_ID=") {
			continue
		}
		base = append(base, e)
	}
	configured := func(bin string) []string {
		return append(append([]string{}, base...),
			"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
			"PUSH_VALIDATION_URL=http://127.0.0.1:8888/internal/push-validation",
			"PUSH_VALIDATION_SECRET=test-secret",
			"PUSH_REPOSITORY_ID=assessments/assessment-1",
		)
	}

	old := strings.Repeat("a", 40)
	next := strings.Repeat("b", 40)
	zero := strings.Repeat("0", 40)

	code, stdout, stderr := runPushHook(t, hook, configured(acceptBin), "refs/heads/main", old, next)
	if code != 0 {
		t.Fatalf("accept: exit %d, stderr %q", code, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("accept was not silent: stdout %q, stderr %q", stdout, stderr)
	}

	code, _, stderr = runPushHook(t, hook, configured(rejectBin), "refs/heads/main", old, next)
	if code == 0 {
		t.Fatalf("reject: hook exited 0, want a failure")
	}
	if !strings.Contains(stderr, "push to main was rejected") || !strings.Contains(stderr, "chapter 1: broken") {
		t.Errorf("reject did not forward the body: %q", stderr)
	}

	code, _, stderr = runPushHook(t, hook, configured(downBin), "refs/heads/main", old, next)
	if code == 0 {
		t.Fatalf("unreachable backend: hook exited 0, want a failure")
	}
	if !strings.Contains(stderr, "push validation is unavailable") {
		t.Errorf("unreachable backend message = %q", stderr)
	}

	// A non-main ref must pass without calling curl at all, so a stub that
	// always fails is the proof.
	code, _, stderr = runPushHook(t, hook, configured(neverBin), "refs/heads/feature", old, next)
	if code != 0 {
		t.Fatalf("feature branch: exit %d, stderr %q", code, stderr)
	}

	code, _, stderr = runPushHook(t, hook, configured(acceptBin), "refs/heads/main", old, zero)
	if code == 0 {
		t.Fatalf("deleting main: hook exited 0, want a failure")
	}
	if !strings.Contains(stderr, "main cannot be deleted") {
		t.Errorf("deleting main message = %q", stderr)
	}

	code, _, stderr = runPushHook(t, hook, base, "refs/heads/main", old, next)
	if code == 0 {
		t.Fatalf("missing configuration: hook exited 0, want a failure")
	}
	if !strings.Contains(stderr, "push validation is not configured") {
		t.Errorf("missing configuration message = %q", stderr)
	}
}

func runPushHook(t *testing.T, hook string, env []string, ref, old, next string) (int, string, string) {
	t.Helper()
	cmd := exec.Command("sh", hook, ref, old, next)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return 0, stdout.String(), stderr.String()
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("run hook: %v", err)
	}
	return exitErr.ExitCode(), stdout.String(), stderr.String()
}

func TestPushValidationHandlerRequiresSecret(t *testing.T) {
	// A nil db proves the credential check happens before any database
	// access: a rejected request must not reach it.
	h := &Handlers{cfg: &config.Config{}, pushSecret: "s3cret"}
	handler := h.PushValidationHandler()
	body := "repository=assessments/assessment-1\nref=refs/heads/main\nnew=" + strings.Repeat("b", 40) + "\n"

	for _, header := range []string{"", "wrong-secret"} {
		req := httptest.NewRequest(http.MethodPost, "/internal/push-validation", strings.NewReader(body))
		if header != "" {
			req.Header.Set("X-Push-Validation", header)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("header %q: status = %d, want 401", header, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "missing or invalid push validation credentials") {
			t.Errorf("header %q: body = %q", header, rec.Body.String())
		}
	}
}
