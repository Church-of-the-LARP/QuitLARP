package assessment

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
}

// runGit runs git in dir and returns its trimmed stdout.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// initBare creates an empty bare repository.
func initBare(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "repo.git")
	runGit(t, "", "init", "--bare", "--quiet", "--initial-branch=main", dir)
	return dir
}

// seededBare commits files in a fresh work tree, pushes them to a new bare
// repository and returns the bare directory and the commit sha.
func seededBare(t *testing.T, files map[string]string) (string, string) {
	t.Helper()
	work := t.TempDir()
	runGit(t, work, "init", "--quiet", "--initial-branch=main")
	runGit(t, work, "config", "user.email", "test@example.com")
	runGit(t, work, "config", "user.name", "Test")
	writeTree(t, work, files)
	runGit(t, work, "add", ".")
	runGit(t, work, "commit", "--quiet", "-m", "init")
	sha := runGit(t, work, "rev-parse", "HEAD")

	bare := initBare(t)
	runGit(t, work, "remote", "add", "origin", bare)
	runGit(t, work, "push", "--quiet", "origin", "main")
	return bare, sha
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestMaterializeExtracts(t *testing.T) {
	requireGit(t)
	files := map[string]string{
		"Dockerfile":                 "FROM alpine\n",
		"chapters/1_a/README.md":     "# a\n",
		"chapters/1_a/sub/helper.ml": "let x = 1\n",
	}
	bare, sha := seededBare(t, files)

	dest := filepath.Join(t.TempDir(), "out")
	if err := Materialize(context.Background(), bare, sha, "", dest); err != nil {
		t.Fatalf("Materialize: %v", err)
	}

	if got := readFile(t, filepath.Join(dest, "Dockerfile")); got != "FROM alpine\n" {
		t.Errorf("Dockerfile = %q", got)
	}
	if got := readFile(t, filepath.Join(dest, "chapters", "1_a", "README.md")); got != "# a\n" {
		t.Errorf("README.md = %q", got)
	}
	if got := readFile(t, filepath.Join(dest, "chapters", "1_a", "sub", "helper.ml")); got != "let x = 1\n" {
		t.Errorf("helper.ml = %q", got)
	}
}

// TestMaterializeQuarantine checks that a commit living only in another
// repository's object store resolves when that store is passed as quarantine.
func TestMaterializeQuarantine(t *testing.T) {
	requireGit(t)
	source, sha := seededBare(t, map[string]string{"file.txt": "from source\n"})
	quarantine := filepath.Join(source, "objects")

	other := initBare(t)
	dest := filepath.Join(t.TempDir(), "out")
	if err := Materialize(context.Background(), other, sha, quarantine, dest); err != nil {
		t.Fatalf("Materialize with quarantine: %v", err)
	}
	if got := readFile(t, filepath.Join(dest, "file.txt")); got != "from source\n" {
		t.Errorf("file.txt = %q", got)
	}

	// Without the quarantine the same sha cannot be resolved.
	empty := filepath.Join(t.TempDir(), "plain")
	if err := Materialize(context.Background(), other, sha, "", empty); err == nil {
		t.Fatal("Materialize without quarantine succeeded, want an error")
	}
}

func TestMaterializeMissingSha(t *testing.T) {
	requireGit(t)
	bare, _ := seededBare(t, map[string]string{"file.txt": "hi\n"})

	dest := filepath.Join(t.TempDir(), "out")
	err := Materialize(context.Background(), bare, "0000000000000000000000000000000000000000", "", dest)
	if err == nil {
		t.Fatal("Materialize succeeded for a missing sha, want an error")
	}
	if !strings.Contains(err.Error(), "git archive") {
		t.Errorf("err = %v, want it to mention git archive", err)
	}
}

func TestQuarantineEnv(t *testing.T) {
	base := []string{"PATH=/bin", "GIT_ALTERNATE_OBJECT_DIRECTORIES=/existing"}

	got := quarantineEnv(base, "/quarantine")
	want := "/quarantine" + string(os.PathListSeparator) + "/existing"
	if v := envValue(got, "GIT_ALTERNATE_OBJECT_DIRECTORIES"); v != want {
		t.Errorf("value = %q, want %q", v, want)
	}
	if len(got) != len(base) {
		t.Errorf("env length = %d, want %d", len(got), len(base))
	}

	plain := quarantineEnv([]string{"PATH=/bin"}, "")
	if envValue(plain, "GIT_ALTERNATE_OBJECT_DIRECTORIES") != "" {
		t.Error("empty quarantine set GIT_ALTERNATE_OBJECT_DIRECTORIES")
	}
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return strings.TrimPrefix(kv, prefix)
		}
	}
	return ""
}

func writeTar(t *testing.T, headers []*tar.Header, bodies [][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for i, hdr := range headers {
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar header %d: %v", i, err)
		}
		if len(bodies[i]) > 0 {
			if _, err := tw.Write(bodies[i]); err != nil {
				t.Fatalf("tar body %d: %v", i, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	return buf.Bytes()
}

func TestExtractTarMembers(t *testing.T) {
	dest := t.TempDir()
	body := []byte("payload")
	data := writeTar(t, []*tar.Header{
		{Name: "dir/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "dir/file.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))},
		{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "dir/file.txt"},
		{Name: "fifo", Typeflag: tar.TypeFifo, Mode: 0o644},
	}, [][]byte{nil, body, nil, nil})

	if err := extractTar(bytes.NewReader(data), dest); err != nil {
		t.Fatalf("extractTar: %v", err)
	}
	if got := readFile(t, filepath.Join(dest, "dir", "file.txt")); got != "payload" {
		t.Errorf("file = %q", got)
	}
	target, err := os.Readlink(filepath.Join(dest, "link"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != "dir/file.txt" {
		t.Errorf("link target = %q", target)
	}
	if _, err := os.Lstat(filepath.Join(dest, "fifo")); !os.IsNotExist(err) {
		t.Error("fifo member was not skipped")
	}
}

func TestExtractTarRejectsEscape(t *testing.T) {
	dest := t.TempDir()
	body := []byte("pwned")
	data := writeTar(t, []*tar.Header{
		{Name: "../evil.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))},
	}, [][]byte{body})

	if err := extractTar(bytes.NewReader(data), dest); err == nil {
		t.Fatal("extractTar accepted a ../ member")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "evil.txt")); !os.IsNotExist(err) {
		t.Fatal("extractTar wrote outside dest")
	}
}

func TestExtractTarRefusesSymlinkedParent(t *testing.T) {
	dest := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dest, "sneak")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	body := []byte("x")
	data := writeTar(t, []*tar.Header{
		{Name: "sneak/file.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))},
	}, [][]byte{body})

	if err := extractTar(bytes.NewReader(data), dest); err == nil {
		t.Fatal("extractTar wrote through a symlinked parent")
	}
	if _, err := os.Stat(filepath.Join(outside, "file.txt")); !os.IsNotExist(err) {
		t.Fatal("extractTar escaped through a symlink")
	}
}

func TestExtractTarReplacesSymlink(t *testing.T) {
	dest := t.TempDir()
	outside := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(outside, []byte("untouched"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	body := []byte("new")
	data := writeTar(t, []*tar.Header{
		{Name: "link", Typeflag: tar.TypeSymlink, Linkname: outside},
		{Name: "link", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))},
	}, [][]byte{nil, body})

	if err := extractTar(bytes.NewReader(data), dest); err != nil {
		t.Fatalf("extractTar: %v", err)
	}
	info, err := os.Lstat(filepath.Join(dest, "link"))
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("link mode = %v, want a regular file", info.Mode())
	}
	if got := readFile(t, filepath.Join(dest, "link")); got != "new" {
		t.Errorf("link content = %q, want new", got)
	}
	if got := readFile(t, outside); got != "untouched" {
		t.Errorf("outside = %q, want untouched", got)
	}
}
