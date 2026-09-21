package repos

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v: %s", name, args, err, out)
	}
}

func initBare(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "repo.git")
	run(t, "", "git", "init", "--bare", "--quiet", "--initial-branch=main", dir)
	return dir
}

// seededRepo builds a bare repo with a nested directory and root files.
func seededRepo(t *testing.T) string {
	t.Helper()
	work := t.TempDir()
	bare := initBare(t)

	run(t, work, "git", "init", "--quiet", "--initial-branch=main")
	run(t, work, "git", "config", "user.email", "test@example.com")
	run(t, work, "git", "config", "user.name", "Test")

	mustWrite(t, filepath.Join(work, "readme.md"), "hello world\n")
	mustWrite(t, filepath.Join(work, "a.txt"), "a\n")
	mustWrite(t, filepath.Join(work, "docs", "guide", "intro.txt"), "nested content\n")

	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "--quiet", "-m", "init")
	run(t, work, "git", "remote", "add", "origin", bare)
	run(t, work, "git", "push", "--quiet", "origin", "main")
	return bare
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
}

func TestTreeOrdering(t *testing.T) {
	requireGit(t)
	r, err := OpenDir(seededRepo(t))
	if err != nil {
		t.Fatalf("OpenDir: %v", err)
	}

	tree, err := r.Tree("", "")
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if tree.Ref != "main" {
		t.Errorf("Ref = %q, want %q", tree.Ref, "main")
	}
	if tree.Empty {
		t.Error("Empty = true, want false")
	}

	got := make([]string, 0, len(tree.Entries))
	for _, e := range tree.Entries {
		got = append(got, e.Name)
	}
	want := []string{"docs", "a.txt", "readme.md"}
	if len(got) != len(want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entries = %v, want %v", got, want)
		}
	}
	if tree.Entries[0].Type != "tree" {
		t.Errorf("docs type = %q, want tree", tree.Entries[0].Type)
	}
	if tree.Entries[1].Type != "blob" {
		t.Errorf("a.txt type = %q, want blob", tree.Entries[1].Type)
	}
	if tree.Entries[1].Size != int64(len("a\n")) {
		t.Errorf("a.txt size = %d, want %d", tree.Entries[1].Size, len("a\n"))
	}
	if len(tree.Entries[1].Hash) != 40 {
		t.Errorf("a.txt hash = %q, want a 40 char sha", tree.Entries[1].Hash)
	}
}

func TestTreeDescends(t *testing.T) {
	requireGit(t)
	r, err := OpenDir(seededRepo(t))
	if err != nil {
		t.Fatalf("OpenDir: %v", err)
	}

	docs, err := r.Tree("", "docs")
	if err != nil {
		t.Fatalf("Tree(docs): %v", err)
	}
	if len(docs.Entries) != 1 || docs.Entries[0].Name != "guide" || docs.Entries[0].Type != "tree" {
		t.Fatalf("docs entries = %+v, want one tree named guide", docs.Entries)
	}

	guide, err := r.Tree("", "docs/guide")
	if err != nil {
		t.Fatalf("Tree(docs/guide): %v", err)
	}
	if len(guide.Entries) != 1 || guide.Entries[0].Name != "intro.txt" {
		t.Fatalf("docs/guide entries = %+v, want one blob named intro.txt", guide.Entries)
	}
	if guide.Path != "docs/guide" {
		t.Errorf("Path = %q, want docs/guide", guide.Path)
	}
}

func TestFileContent(t *testing.T) {
	requireGit(t)
	r, err := OpenDir(seededRepo(t))
	if err != nil {
		t.Fatalf("OpenDir: %v", err)
	}

	f, err := r.File("", "docs/guide/intro.txt")
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if f.Content != "nested content\n" {
		t.Errorf("Content = %q, want %q", f.Content, "nested content\n")
	}
	if f.Binary || f.Truncated {
		t.Errorf("Binary = %v, Truncated = %v, want false, false", f.Binary, f.Truncated)
	}
	if f.Size != int64(len("nested content\n")) {
		t.Errorf("Size = %d, want %d", f.Size, len("nested content\n"))
	}
	if f.Ref != "main" {
		t.Errorf("Ref = %q, want main", f.Ref)
	}
}

func TestTreeAndFileErrors(t *testing.T) {
	requireGit(t)
	r, err := OpenDir(seededRepo(t))
	if err != nil {
		t.Fatalf("OpenDir: %v", err)
	}

	if _, err := r.Tree("nope", ""); !errors.Is(err, ErrRefNotFound) {
		t.Errorf("Tree unknown ref err = %v, want ErrRefNotFound", err)
	}
	if _, err := r.Tree("", "nope"); !errors.Is(err, ErrPathNotFound) {
		t.Errorf("Tree unknown path err = %v, want ErrPathNotFound", err)
	}
	if _, err := r.Tree("", "readme.md"); !errors.Is(err, ErrPathNotFound) {
		t.Errorf("Tree on a file err = %v, want ErrPathNotFound", err)
	}
	if _, err := r.File("nope", "readme.md"); !errors.Is(err, ErrRefNotFound) {
		t.Errorf("File unknown ref err = %v, want ErrRefNotFound", err)
	}
	if _, err := r.File("", "nope.txt"); !errors.Is(err, ErrPathNotFound) {
		t.Errorf("File missing err = %v, want ErrPathNotFound", err)
	}
	if _, err := r.File("", "docs"); !errors.Is(err, ErrPathNotFound) {
		t.Errorf("File on a directory err = %v, want ErrPathNotFound", err)
	}
	if _, err := r.File("", ""); !errors.Is(err, ErrPathNotFound) {
		t.Errorf("File on the root err = %v, want ErrPathNotFound", err)
	}
}

func TestEmptyRepoReportsEmpty(t *testing.T) {
	requireGit(t)
	r, err := OpenDir(initBare(t))
	if err != nil {
		t.Fatalf("OpenDir: %v", err)
	}

	tree, err := r.Tree("", "")
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if !tree.Empty {
		t.Error("Empty = false, want true")
	}
	if len(tree.Entries) != 0 {
		t.Errorf("entries = %v, want none", tree.Entries)
	}
	if _, err := r.File("", "readme.md"); !errors.Is(err, ErrRefNotFound) {
		t.Errorf("File err = %v, want ErrRefNotFound", err)
	}
}

func TestOpenDirMissingDoesNotCreate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing.git")
	if _, err := OpenDir(dir); !errors.Is(err, ErrRepoMissing) {
		t.Fatalf("OpenDir err = %v, want ErrRepoMissing", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("OpenDir created %s (stat err = %v), want it left alone", dir, err)
	}
}

func TestIsBinary(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty", nil, false},
		{"text", []byte("hello\n"), false},
		{"utf8", []byte("héllo"), false},
		{"nul", []byte("a\x00b"), true},
		{"invalid utf8", []byte{0xff, 0xfe, 0x01}, true},
	}
	for _, tc := range cases {
		if got := isBinary(tc.data); got != tc.want {
			t.Errorf("isBinary(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}

	// A NUL byte past the sniffed prefix is not seen.
	long := append([]byte("text"), bytes.Repeat([]byte("a"), sniffBytes+10)...)
	long = append(long, 0)
	if isBinary(long) {
		t.Error("isBinary reported a NUL past the sniff prefix as binary")
	}
}
