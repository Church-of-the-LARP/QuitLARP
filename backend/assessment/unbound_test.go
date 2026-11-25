package assessment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeUnboundTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLoadUnboundTree(t *testing.T) {
	root := writeUnboundTree(t, map[string]string{
		"assessment.json":   `{"kind":"unbound"}`,
		"README.md":         "# Build a kanban board\n",
		"deploy/index.html": "<!doctype html><h1>app</h1>",
		"deploy/app.js":     "console.log('app')",
	})

	content, violations := Load(root)
	if len(violations) > 0 {
		t.Fatalf("violations = %v", violations)
	}
	if content.Kind != kindUnbound {
		t.Fatalf("kind = %q", content.Kind)
	}
	if content.DeployDir != "deploy" {
		t.Fatalf("deploy dir = %q", content.DeployDir)
	}
	if len(content.Chapters) != 0 {
		t.Fatalf("chapters = %d, want 0", len(content.Chapters))
	}
	if !strings.Contains(content.Description, "kanban") {
		t.Fatalf("description = %q", content.Description)
	}
}

func TestLoadUnboundRequiresEntryPoint(t *testing.T) {
	root := writeUnboundTree(t, map[string]string{
		"assessment.json": `{"kind":"unbound"}`,
		"deploy/app.js":   "console.log('app')",
	})
	_, violations := Load(root)
	if !hasViolation(violations, "deploy/index.html: missing") {
		t.Fatalf("violations = %v", violations)
	}
}

func TestLoadUnboundRejectsSymlinks(t *testing.T) {
	root := writeUnboundTree(t, map[string]string{
		"assessment.json":   `{"kind":"unbound"}`,
		"deploy/index.html": "<!doctype html>",
		"outside.txt":       "secret",
	})
	if err := os.Symlink("../outside.txt", filepath.Join(root, "deploy", "link.txt")); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	_, violations := Load(root)
	if !hasViolation(violations, "symlinks are not allowed") {
		t.Fatalf("violations = %v", violations)
	}
}

func TestLoadUnboundAcceptsEventWorkspaceWithoutManifest(t *testing.T) {
	root := writeUnboundTree(t, map[string]string{
		"deploy/index.html": "<!doctype html>",
	})
	content, violations := LoadUnbound(root)
	if len(violations) > 0 {
		t.Fatalf("violations = %v", violations)
	}
	if content.Kind != kindUnbound {
		t.Fatalf("kind = %q", content.Kind)
	}
}

func TestLoadUnboundRejectsLeetcodeManifest(t *testing.T) {
	root := writeUnboundTree(t, map[string]string{
		"assessment.json":   `{"kind":"leetcode"}`,
		"deploy/index.html": "<!doctype html>",
	})
	_, violations := LoadUnbound(root)
	if !hasViolation(violations, "kind must be") {
		t.Fatalf("violations = %v", violations)
	}
}

func hasViolation(violations []string, fragment string) bool {
	for _, violation := range violations {
		if strings.Contains(violation, fragment) {
			return true
		}
	}
	return false
}
