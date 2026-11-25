package repos

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func testAuthor() object.Signature {
	return object.Signature{
		Name:  "solver",
		Email: "solver@example.com",
		When:  time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC),
	}
}

func readBlob(t *testing.T, repo *git.Repository, commit *object.Commit, path string) string {
	t.Helper()
	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("commit tree: %v", err)
	}
	entry, err := tree.FindEntry(path)
	if err != nil {
		t.Fatalf("find %s: %v", path, err)
	}
	blob, err := repo.BlobObject(entry.Hash)
	if err != nil {
		t.Fatalf("blob %s: %v", path, err)
	}
	reader, err := blob.Reader()
	if err != nil {
		t.Fatalf("blob reader %s: %v", path, err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read blob %s: %v", path, err)
	}
	return string(data)
}

func TestCommitFilesCreatesRepository(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "solutions", "assessment-1", "user-2.git")
	files := map[string][]byte{
		"chapters/1_two-sum/glue/two_sum_functions.py": []byte("def two_sum(nums, target):\n    return [0, 1]\n"),
		"README.md": []byte("solution\n"),
	}

	sha, err := CommitFiles(dir, files, "solution: assessment 1", testAuthor())
	if err != nil {
		t.Fatalf("CommitFiles: %v", err)
	}

	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ref, err := repo.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		t.Fatalf("main ref: %v", err)
	}
	if ref.Hash().String() != sha {
		t.Fatalf("main points at %s, want %s", ref.Hash(), sha)
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if len(commit.ParentHashes) != 0 {
		t.Fatalf("first commit has %d parents, want 0", len(commit.ParentHashes))
	}
	got := readBlob(t, repo, commit, "chapters/1_two-sum/glue/two_sum_functions.py")
	if want := string(files["chapters/1_two-sum/glue/two_sum_functions.py"]); got != want {
		t.Fatalf("nested blob = %q, want %q", got, want)
	}
	if got := readBlob(t, repo, commit, "README.md"); got != "solution\n" {
		t.Fatalf("root blob = %q", got)
	}
}

func TestCommitFilesMovesForward(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "user-2.git")
	first, err := CommitFiles(dir, map[string][]byte{"a.py": []byte("one\n")}, "first", testAuthor())
	if err != nil {
		t.Fatalf("first commit: %v", err)
	}
	second, err := CommitFiles(dir, map[string][]byte{"a.py": []byte("two\n")}, "second", testAuthor())
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}
	if first == second {
		t.Fatal("second commit did not move the branch")
	}

	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ref, err := repo.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		t.Fatalf("main ref: %v", err)
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if len(commit.ParentHashes) != 1 || commit.ParentHashes[0].String() != first {
		t.Fatalf("parents = %v, want [%s]", commit.ParentHashes, first)
	}
	if got := readBlob(t, repo, commit, "a.py"); got != "two\n" {
		t.Fatalf("blob = %q, want %q", got, "two\n")
	}
}

func TestCommitFilesRejectsBadPaths(t *testing.T) {
	cases := []struct {
		name  string
		files map[string][]byte
	}{
		{"absolute", map[string][]byte{"/etc/passwd": []byte("x")}},
		{"escape", map[string][]byte{"../outside": []byte("x")}},
		{"empty segment", map[string][]byte{"a//b": []byte("x")}},
		{"nested under file", map[string][]byte{"a": []byte("x"), "a/b": []byte("y")}},
		{"file over directory", map[string][]byte{"a/b": []byte("y"), "a": []byte("x")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "user-2.git")
			if _, err := CommitFiles(dir, tc.files, "bad", testAuthor()); err == nil {
				t.Fatal("CommitFiles accepted an invalid file set")
			}
		})
	}
}

func TestCommitFilePreservesOtherFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "workspace.git")
	if _, err := CommitFiles(dir, map[string][]byte{
		"deploy/index.html": []byte("<h1>one</h1>"),
		"README.md":         []byte("readme"),
	}, "seed", testAuthor()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sha, err := CommitFile(dir, "deploy/index.html", []byte("<h1>two</h1>"), "edit", testAuthor())
	if err != nil {
		t.Fatalf("CommitFile: %v", err)
	}
	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := repo.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		t.Fatal(err)
	}
	if got := readBlob(t, repo, commit, "deploy/index.html"); got != "<h1>two</h1>" {
		t.Fatalf("edited blob = %q", got)
	}
	if got := readBlob(t, repo, commit, "README.md"); got != "readme" {
		t.Fatalf("preserved blob = %q", got)
	}
}

func TestDeleteFileRemovesOnlyThatFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "workspace.git")
	if _, err := CommitFiles(dir, map[string][]byte{
		"deploy/index.html": []byte("<h1>one</h1>"),
		"deploy/app.js":     []byte("app"),
	}, "seed", testAuthor()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sha, err := DeleteFile(dir, "deploy/app.js", "delete", testAuthor())
	if err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := repo.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commit.Tree(); err != nil {
		t.Fatal(err)
	}
	tree, _ := commit.Tree()
	if _, err := tree.FindEntry("deploy/app.js"); err == nil {
		t.Fatal("deleted file is still in the tree")
	}
	if got := readBlob(t, repo, commit, "deploy/index.html"); got != "<h1>one</h1>" {
		t.Fatalf("preserved blob = %q", got)
	}
}
