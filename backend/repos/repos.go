// Package repos reads the bare git repositories that back assessments. It
// never creates or modifies them.
package repos

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	// maxContentBytes caps how much of a file is returned.
	maxContentBytes = 1 << 20
	// sniffBytes is the prefix inspected to tell text from binary.
	sniffBytes = 8000
)

// Errors the handlers map to HTTP statuses.
var (
	ErrRepoMissing  = errors.New("repository does not exist")
	ErrRefNotFound  = errors.New("revision not found")
	ErrPathNotFound = errors.New("path not found")
)

// Entry is one row of a directory listing.
type Entry struct {
	Name string
	Type string // "tree" or "blob"
	Size int64
	Hash string
}

// Tree is a directory listing at a resolved revision.
type Tree struct {
	Ref     string
	Path    string
	Entries []Entry
	Empty   bool
}

// File is a single file at a resolved revision. Content is empty for binary
// or truncated files.
type File struct {
	Ref       string
	Path      string
	Size      int64
	Hash      string
	Binary    bool
	Truncated bool
	Content   string
}

// Repo is a read-only handle on a bare repository.
type Repo struct {
	repo *git.Repository
	head string // short name of the HEAD branch, "" while it is unborn
}

// OpenDir opens a bare repository for reading; it never initializes one.
func OpenDir(dir string) (*Repo, error) {
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrRepoMissing, dir)
	}
	repo, err := git.PlainOpen(dir)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, fmt.Errorf("%w: %s", ErrRepoMissing, dir)
		}
		return nil, err
	}
	return &Repo{repo: repo, head: headBranch(repo)}, nil
}

// Tree lists the directory at path in ref; an empty ref means the HEAD branch.
// A repository with no commits reports Empty.
func (r *Repo) Tree(ref, path string) (*Tree, error) {
	path = cleanPath(path)
	ref = strings.TrimSpace(ref)
	if ref == "" {
		if r.head == "" {
			return &Tree{Entries: []Entry{}, Empty: true}, nil
		}
		ref = r.head
	}

	root, err := r.treeAt(ref)
	if err != nil {
		return nil, err
	}
	dir, err := r.subtree(root, path)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(dir.Entries))
	for _, e := range dir.Entries {
		entries = append(entries, Entry{
			Name: e.Name,
			Type: entryType(e.Mode),
			Size: r.size(e),
			Hash: e.Hash.String(),
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if (entries[i].Type == "tree") != (entries[j].Type == "tree") {
			return entries[i].Type == "tree"
		}
		return entries[i].Name < entries[j].Name
	})

	return &Tree{Ref: ref, Path: path, Entries: entries}, nil
}

// File reads the file at path in ref; an empty ref means the HEAD branch.
func (r *Repo) File(ref, path string) (*File, error) {
	path = cleanPath(path)
	if path == "" {
		return nil, fmt.Errorf("%w: %q", ErrPathNotFound, path)
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		if r.head == "" {
			return nil, fmt.Errorf("%w: %s", ErrRefNotFound, "empty repository")
		}
		ref = r.head
	}

	root, err := r.treeAt(ref)
	if err != nil {
		return nil, err
	}
	entry, err := root.FindEntry(path)
	if err != nil || !entry.Mode.IsFile() {
		return nil, fmt.Errorf("%w: %s", ErrPathNotFound, path)
	}
	blob, err := r.repo.BlobObject(entry.Hash)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrPathNotFound, path)
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, maxContentBytes+1))
	if err != nil {
		return nil, err
	}
	truncated := len(data) > maxContentBytes
	if truncated {
		data = data[:maxContentBytes]
	}

	out := &File{
		Ref:       ref,
		Path:      path,
		Size:      blob.Size,
		Hash:      entry.Hash.String(),
		Binary:    isBinary(data),
		Truncated: truncated,
	}
	if !out.Binary && !out.Truncated {
		out.Content = string(data)
	}
	return out, nil
}

// headBranch returns the branch HEAD resolves to, or "" while it is unborn.
func headBranch(repo *git.Repository) string {
	ref, err := repo.Reference(plumbing.HEAD, false)
	if err != nil {
		return ""
	}
	switch ref.Type() {
	case plumbing.SymbolicReference:
		name := ref.Target().Short()
		if _, err := repo.ResolveRevision(plumbing.Revision(name)); err != nil {
			return ""
		}
		return name
	case plumbing.HashReference:
		// Detached HEAD: browse the commit it points at.
		if _, err := repo.ResolveRevision(plumbing.Revision("HEAD")); err != nil {
			return ""
		}
		return "HEAD"
	}
	return ""
}

// treeAt resolves ref to a tree, accepting commits and bare trees.
func (r *Repo) treeAt(ref string) (*object.Tree, error) {
	hash, err := r.repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrRefNotFound, ref)
	}
	if commit, err := r.repo.CommitObject(*hash); err == nil {
		tree, err := commit.Tree()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrRefNotFound, ref)
		}
		return tree, nil
	}
	if tree, err := r.repo.TreeObject(*hash); err == nil {
		return tree, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrRefNotFound, ref)
}

func (r *Repo) subtree(root *object.Tree, path string) (*object.Tree, error) {
	cur := root
	for _, seg := range strings.Split(path, "/") {
		if seg == "" {
			continue
		}
		entry, err := cur.FindEntry(seg)
		if err != nil || entry.Mode != filemode.Dir {
			return nil, fmt.Errorf("%w: %s", ErrPathNotFound, path)
		}
		next, err := r.repo.TreeObject(entry.Hash)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrPathNotFound, path)
		}
		cur = next
	}
	return cur, nil
}

func (r *Repo) size(entry object.TreeEntry) int64 {
	if entry.Mode == filemode.Dir || entry.Mode == filemode.Submodule {
		return 0
	}
	blob, err := r.repo.BlobObject(entry.Hash)
	if err != nil {
		return 0
	}
	return blob.Size
}

func entryType(mode filemode.FileMode) string {
	if mode == filemode.Dir {
		return "tree"
	}
	return "blob"
}

// isBinary sniffs the leading bytes for a NUL byte or invalid UTF-8.
func isBinary(data []byte) bool {
	if len(data) > sniffBytes {
		data = data[:sniffBytes]
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return true
	}
	return !utf8.Valid(data)
}

func cleanPath(path string) string {
	return strings.Trim(strings.TrimSpace(path), "/")
}
