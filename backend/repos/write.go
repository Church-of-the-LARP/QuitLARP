package repos

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// solutionBranch is the only branch a written repository carries.
const solutionBranch = "main"

// CommitFiles writes files, keyed by slash path relative to the repository
// root, into the bare repository at dir, creating the repository when it does
// not exist yet, and returns the new commit id. The branch only moves
// forward: the new commit has the current branch tip as its parent.
func CommitFiles(dir string, files map[string][]byte, message string, author object.Signature) (string, error) {
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
