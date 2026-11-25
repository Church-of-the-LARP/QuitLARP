package repos

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// solutionBranch is the only branch a written repository carries.
const solutionBranch = "main"

// maxCommitBytes caps the total size of a tree this package reads before
// rewriting it. Web app workspaces are small; the cap keeps a crafted
// repository from exhausting memory.
const maxCommitBytes = 16 << 20

// writeMu serializes every commit. CommitFile rewrites the whole tree from the
// current main tip, so two concurrent commits in one repository would
// otherwise race and lose one of the trees.
var writeMu sync.Mutex

// CommitFiles writes files, keyed by slash path relative to the repository
// root, into the bare repository at dir, creating the repository when it does
// not exist yet, and returns the new commit id. The branch only moves
// forward: the new commit has the current branch tip as its parent.
func CommitFiles(dir string, files map[string][]byte, message string, author object.Signature) (string, error) {
	writeMu.Lock()
	defer writeMu.Unlock()
	return commitFiles(dir, files, message, author)
}

func commitFiles(dir string, files map[string][]byte, message string, author object.Signature) (string, error) {
	if len(files) == 0 {
		return "", errors.New("no files to commit")
	}
	root := &pathNode{children: map[string]*pathNode{}}
	for path, content := range files {
		parts, err := splitCleanPath(path)
		if err != nil {
			return "", err
		}
		node := root
		for _, part := range parts {
			if node.file {
				return "", fmt.Errorf("path %q is nested under a file", path)
			}
			child, ok := node.children[part]
			if !ok {
				child = &pathNode{children: map[string]*pathNode{}}
				node.children[part] = child
			}
			node = child
		}
		if node.file {
			return "", fmt.Errorf("duplicate path %q", path)
		}
		if len(node.children) > 0 {
			return "", fmt.Errorf("path %q conflicts with a directory", path)
		}
		node.file = true
		node.content = content
	}

	repo, err := openOrInitBare(dir)
	if err != nil {
		return "", err
	}
	treeHash, err := writeNode(repo.Storer, root)
	if err != nil {
		return "", err
	}

	var parents []plumbing.Hash
	tip, err := repo.Reference(plumbing.NewBranchReferenceName(solutionBranch), true)
	switch {
	case err == nil:
		parents = append(parents, tip.Hash())
	case !errors.Is(err, plumbing.ErrReferenceNotFound):
		return "", err
	}

	commit := &object.Commit{
		Author:       author,
		Committer:    author,
		Message:      message,
		TreeHash:     treeHash,
		ParentHashes: parents,
	}
	obj := repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.CommitObject)
	if err := commit.Encode(obj); err != nil {
		return "", err
	}
	hash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", err
	}
	if err := repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName(solutionBranch), hash)); err != nil {
		return "", err
	}
	return hash.String(), nil
}

// IsBinary reports whether data looks like a binary file (NUL or invalid
// UTF-8), the same rule tree and file browsing uses.
func IsBinary(data []byte) bool {
	return isBinary(data)
}

// SnapshotFiles returns every regular file of main, keyed by slash path. It is
// used to initialize a browser workspace from git.
func SnapshotFiles(dir string) (map[string][]byte, error) {
	writeMu.Lock()
	defer writeMu.Unlock()
	return filesOnMain(dir)
}

// CommitChanges applies a set of file writes and deletions to main in one
// commit, preserving every other file in the tree.
func CommitChanges(dir string, files map[string][]byte, deletes []string, message string, author object.Signature) (string, error) {
	for path := range files {
		if _, err := splitCleanPath(path); err != nil {
			return "", err
		}
	}
	for _, path := range deletes {
		if _, err := splitCleanPath(path); err != nil {
			return "", err
		}
	}
	writeMu.Lock()
	defer writeMu.Unlock()

	current, err := filesOnMain(dir)
	if err != nil {
		return "", err
	}
	for path, content := range files {
		current[path] = content
	}
	for _, path := range deletes {
		delete(current, path)
	}
	if len(current) == 0 {
		return "", errors.New("cannot commit an empty workspace")
	}
	return commitFiles(dir, current, message, author)
}

// CommitFile replaces one file on main while preserving every other file in
// the branch tip. It is the write path behind browser edits.
func CommitFile(dir, filePath string, content []byte, message string, author object.Signature) (string, error) {
	if _, err := splitCleanPath(filePath); err != nil {
		return "", err
	}
	writeMu.Lock()
	defer writeMu.Unlock()

	files, err := filesOnMain(dir)
	if err != nil {
		return "", err
	}
	files[filePath] = content
	return commitFiles(dir, files, message, author)
}

// DeleteFile removes one file from main while preserving the rest of the tree.
func DeleteFile(dir, filePath, message string, author object.Signature) (string, error) {
	if _, err := splitCleanPath(filePath); err != nil {
		return "", err
	}
	writeMu.Lock()
	defer writeMu.Unlock()

	files, err := filesOnMain(dir)
	if err != nil {
		return "", err
	}
	if _, ok := files[filePath]; !ok {
		return "", fmt.Errorf("%w: %s", ErrPathNotFound, filePath)
	}
	delete(files, filePath)
	if len(files) == 0 {
		return "", errors.New("cannot delete the last file of the repository")
	}
	return commitFiles(dir, files, message, author)
}

// CommitDir copies every regular file under root into dir's main branch as
// one commit. It is used to seed an event workspace from a task repository.
func CommitDir(dir, root, message string, author object.Signature) (string, error) {
	files := map[string][]byte{}
	var total int64
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if total+info.Size() > maxCommitBytes {
			return fmt.Errorf("repository content exceeds %d bytes", maxCommitBytes)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		total += int64(len(data))
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", errors.New("no files to commit")
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	return commitFiles(dir, files, message, author)
}

// filesOnMain returns every regular file of the main branch tip, keyed by
// slash path. A repository with no commits yields an empty map.
func filesOnMain(dir string) (map[string][]byte, error) {
	files := map[string][]byte{}
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}
		return nil, err
	}
	repo, err := git.PlainOpen(dir)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return files, nil
		}
		return nil, err
	}
	ref, err := repo.Reference(plumbing.NewBranchReferenceName(solutionBranch), true)
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return files, nil
	}
	if err != nil {
		return nil, err
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var total int64
	iter := tree.Files()
	err = iter.ForEach(func(f *object.File) error {
		if f.Mode == filemode.Symlink || f.Mode == filemode.Submodule {
			return nil
		}
		if total+f.Size > maxCommitBytes {
			return fmt.Errorf("repository content exceeds %d bytes", maxCommitBytes)
		}
		contents, err := f.Contents()
		if err != nil {
			return err
		}
		total += int64(len(contents))
		files[f.Name] = []byte(contents)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// pathNode is one level of the tree a file set builds.
type pathNode struct {
	children map[string]*pathNode
	file     bool
	content  []byte
}

// writeNode writes the tree of a node depth-first and returns its hash. A
// node that carries a file is a leaf and never has children.
func writeNode(st storer.EncodedObjectStorer, node *pathNode) (plumbing.Hash, error) {
	if node.file {
		return writeBlob(st, node.content)
	}
	entries := make([]object.TreeEntry, 0, len(node.children))
	for name, child := range node.children {
		hash, err := writeNode(st, child)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		mode := filemode.Dir
		if child.file {
			mode = filemode.Regular
		}
		entries = append(entries, object.TreeEntry{Name: name, Mode: mode, Hash: hash})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	tree := &object.Tree{Entries: entries}
	obj := st.NewEncodedObject()
	obj.SetType(plumbing.TreeObject)
	if err := tree.Encode(obj); err != nil {
		return plumbing.ZeroHash, err
	}
	return st.SetEncodedObject(obj)
}

func writeBlob(st storer.EncodedObjectStorer, content []byte) (plumbing.Hash, error) {
	obj := st.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	w, err := obj.Writer()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	if _, err := w.Write(content); err != nil {
		_ = w.Close()
		return plumbing.ZeroHash, err
	}
	if err := w.Close(); err != nil {
		return plumbing.ZeroHash, err
	}
	return st.SetEncodedObject(obj)
}

// openOrInitBare opens the bare repository at dir or creates it with main as
// the initial branch.
func openOrInitBare(dir string) (*git.Repository, error) {
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err == nil {
		return git.PlainOpen(dir)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return nil, fmt.Errorf("create repository dir: %w", err)
	}
	repo, err := git.PlainInit(dir, true)
	if err != nil {
		return nil, fmt.Errorf("init repository %s: %w", dir, err)
	}
	if err := repo.Storer.SetReference(
		plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(solutionBranch))); err != nil {
		return nil, fmt.Errorf("point repository HEAD at %s: %w", solutionBranch, err)
	}
	return repo, nil
}

// splitCleanPath splits a slash path into segments, refusing anything that
// could escape the repository root.
func splitCleanPath(path string) ([]string, error) {
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "\\") {
		return nil, fmt.Errorf("invalid repository path %q", path)
	}
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return nil, fmt.Errorf("invalid repository path %q", path)
		}
	}
	return parts, nil
}
