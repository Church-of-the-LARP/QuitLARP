// Package deployments publishes the deploy/ directory of an event workspace
// as a static site. Every event task has an unguessable public id and is
// served under /deploy/<id>/, so the host can look at the running app next to
// the code while participants keep editing and pushing.
package deployments

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"backend/assessment"
)

// ErrNotDeployable means the commit has no deploy/index.html entry point.
var ErrNotDeployable = errors.New("the repository has no deploy/index.html")

// publicIDPattern matches the ids generated for event deployments.
var publicIDPattern = regexp.MustCompile(`^[a-z0-9]{16,40}$`)

// Manager owns the on-disk deployment trees.
type Manager struct {
	root string
	mu   sync.Mutex
}

// NewManager builds a manager rooted at dir (DEPLOYMENTS_DIR).
func NewManager(dir string) *Manager {
	return &Manager{root: dir}
}

// Deploy materializes sha from repoDir and publishes its deploy/ directory as
// the current version of publicID. It is safe to call concurrently; the
// current pointer only moves after a complete copy exists.
func (m *Manager) Deploy(ctx context.Context, publicID, repoDir, sha string) error {
	if !publicIDPattern.MatchString(publicID) {
		return fmt.Errorf("invalid deployment id %q", publicID)
	}
	if sha == "" {
		return errors.New("no commit to deploy")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	base := filepath.Join(m.root, publicID)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp(base, ".build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	if err := assessment.Materialize(ctx, repoDir, sha, "", tmp); err != nil {
		return fmt.Errorf("materialize %s: %w", sha, err)
	}
	src := filepath.Join(tmp, "deploy")
	if info, err := os.Stat(filepath.Join(src, "index.html")); err != nil || !info.Mode().IsRegular() {
		return ErrNotDeployable
	}

	target := filepath.Join(base, sha)
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	if err := copyVerified(src, target); err != nil {
		os.RemoveAll(target)
		return fmt.Errorf("publish deploy/: %w", err)
	}
	if info, err := os.Stat(filepath.Join(target, "index.html")); err != nil || !info.Mode().IsRegular() {
		os.RemoveAll(target)
		return ErrNotDeployable
	}
	if err := writeCurrent(base, sha); err != nil {
		return err
	}
	m.cleanup(base, sha)
	return nil
}

// Current returns the commit currently served for publicID.
func (m *Manager) Current(publicID string) (string, bool) {
	if !publicIDPattern.MatchString(publicID) {
		return "", false
	}
	data, err := os.ReadFile(filepath.Join(m.root, publicID, "current"))
	if err != nil {
		return "", false
	}
	sha := strings.TrimSpace(string(data))
	if sha == "" {
		return "", false
	}
	return sha, true
}

// Handler serves /deploy/<publicId>/… from the current version. Unknown
// paths and extension-less misses fall back to index.html so client-side
// routed apps work.
func (m *Manager) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rest, ok := strings.CutPrefix(r.URL.Path, "/deploy/")
		if !ok {
			http.NotFound(w, r)
			return
		}
		publicID, tail, _ := strings.Cut(rest, "/")
		if tail == "" && !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
			return
		}
		sha, ok := m.Current(publicID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		base := filepath.Join(m.root, publicID, sha)
		file, ok := resolveFile(base, tail)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Deployed apps are participant-controlled. Force an opaque origin in
		// every browser context so their scripts cannot read the host's session
		// cookie or call the API as the viewer. Relative fetches still work:
		// the wildcard CORS header below lets the opaque origin load them.
		w.Header().Set("Content-Security-Policy", "sandbox allow-scripts allow-forms allow-popups allow-modals")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if strings.HasSuffix(file, "index.html") {
			w.Header().Set("Cache-Control", "no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=60")
		}
		http.ServeFile(w, r, file)
	})
}

// resolveFile maps a URL path under the deployment root to a file on disk,
// refusing traversal and falling back to index.html for SPA routes.
func resolveFile(base, tail string) (string, bool) {
	rel := path.Clean("/" + tail)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "." {
		rel = ""
	}
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	file := filepath.Join(base, filepath.FromSlash(rel))
	if !within(base, file) {
		return "", false
	}
	info, err := os.Stat(file)
	if err == nil && info.IsDir() {
		file = filepath.Join(file, "index.html")
		info, err = os.Stat(file)
	}
	if err != nil {
		if os.IsNotExist(err) && path.Ext(rel) == "" {
			file = filepath.Join(base, "index.html")
			if info, statErr := os.Stat(file); statErr == nil && info.Mode().IsRegular() {
				return file, true
			}
		}
		return "", false
	}
	if !info.Mode().IsRegular() {
		return "", false
	}
	return file, true
}

// copyVerified copies regular files and directories only. Symlinks are
// skipped: a deployment is served by the backend, so a link could otherwise
// read files outside the published tree.
func copyVerified(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// writeCurrent atomically points publicID at sha.
func writeCurrent(base, sha string) error {
	tmp, err := os.CreateTemp(base, ".current-")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := io.WriteString(tmp, sha); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	return os.Rename(name, filepath.Join(base, "current"))
}

// cleanup removes old version directories. A request that already resolved a
// previous current may still be reading it, but deploys are infrequent and a
// 404 on that request is preferable to unbounded disk growth.
func (m *Manager) cleanup(base, keep string) {
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || name == keep || strings.HasPrefix(name, ".") {
			continue
		}
		os.RemoveAll(filepath.Join(base, name))
	}
}

func within(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
