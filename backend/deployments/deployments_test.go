package deployments

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"

	"backend/repos"
)

func TestDeployPublishesOnlyTheDeployDirectory(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(t.TempDir(), "app.git")
	sha, err := repos.CommitFiles(repoDir, map[string][]byte{
		"README.md":         []byte("readme"),
		"deploy/index.html": []byte("<h1>app</h1>"),
		"deploy/app.js":     []byte("console.log('app')"),
	}, "seed", object.Signature{Name: "test", Email: "test@example.com", When: time.Now()})
	if err != nil {
		t.Fatalf("seed repository: %v", err)
	}

	manager := NewManager(root)
	const id = "0123456789abcdef"
	if err := manager.Deploy(context.Background(), id, repoDir, sha); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	current, ok := manager.Current(id)
	if !ok || current != sha {
		t.Fatalf("current = %q, %v", current, ok)
	}
	index := filepath.Join(root, id, sha, "index.html")
	if data, err := os.ReadFile(index); err != nil || string(data) != "<h1>app</h1>" {
		t.Fatalf("index = %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(root, id, sha, "README.md")); !os.IsNotExist(err) {
		t.Fatalf("README should not be published: %v", err)
	}

	recorder := httptest.NewRecorder()
	manager.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/deploy/"+id+"/", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "<h1>app</h1>" {
		t.Fatalf("serve index: %d %q", recorder.Code, recorder.Body.String())
	}

	// Client-side routes fall back to index.html.
	recorder = httptest.NewRecorder()
	manager.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/deploy/"+id+"/board/42", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "<h1>app</h1>" {
		t.Fatalf("serve fallback: %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestDeployRejectsMissingEntryPoint(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "app.git")
	sha, err := repos.CommitFiles(repoDir, map[string][]byte{
		"deploy/app.js": []byte("console.log('app')"),
	}, "seed", object.Signature{Name: "test", Email: "test@example.com", When: time.Now()})
	if err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	manager := NewManager(t.TempDir())
	if err := manager.Deploy(context.Background(), "0123456789abcdef", repoDir, sha); err != ErrNotDeployable {
		t.Fatalf("deploy error = %v, want %v", err, ErrNotDeployable)
	}
}

func TestResolveFileRefusesTraversal(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "index.html"), []byte("index"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "app.js"), []byte("app"), 0o644); err != nil {
		t.Fatal(err)
	}

	file, ok := resolveFile(base, "../../etc/passwd")
	if ok && !within(base, file) {
		t.Fatalf("traversal escaped to %q", file)
	}
	file, ok = resolveFile(base, "missing/route")
	if !ok || filepath.Base(file) != "index.html" {
		t.Fatalf("fallback = %q, %v", file, ok)
	}
	file, ok = resolveFile(base, "app.js")
	if !ok || filepath.Base(file) != "app.js" {
		t.Fatalf("asset = %q, %v", file, ok)
	}
}
