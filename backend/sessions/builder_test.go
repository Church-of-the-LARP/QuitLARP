package sessions

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func contextNames(t *testing.T, trees []contextTree, extra map[string][]byte) map[string]bool {
	t.Helper()
	var buf bytes.Buffer
	if err := writeContext(&buf, trees, extra); err != nil {
		t.Fatalf("writeContext: %v", err)
	}
	names := map[string]bool{}
	tr := tar.NewReader(&buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read context: %v", err)
		}
		names[hdr.Name] = true
	}
	return names
}

func TestWriteContextBuildInclusion(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "_build", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_build", "x", "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dune-project"), []byte("(lang dune 3.0)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	skipped := contextNames(t, []contextTree{{Root: root, Prefix: "tree"}}, nil)
	if skipped["tree/_build/x/a.txt"] {
		t.Error("_build should be skipped unless the tree asks for it")
	}

	kept := contextNames(t, []contextTree{{Root: root, Prefix: "tree", IncludeBuild: true}},
		map[string][]byte{"Dockerfile": []byte("FROM scratch\n")})
	if !kept["tree/_build/x/a.txt"] {
		t.Errorf("_build should be part of the context; members: %v", kept)
	}
	if !kept["Dockerfile"] {
		t.Errorf("extra files should sit at the context root; members: %v", kept)
	}
}
