package supervisor

import (
	"archive/tar"
	"bytes"
	"io"
	"strings"
	"testing"
)

func tarNames(t *testing.T, raw []byte) map[string]bool {
	t.Helper()
	names := make(map[string]bool)
	tr := tar.NewReader(bytes.NewReader(raw))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read context tar: %v", err)
		}
		names[hdr.Name] = true
	}
	return names
}

func TestHarnessDockerfileEmbedded(t *testing.T) {
	body := HarnessDockerfile()
	if len(body) == 0 {
		t.Fatal("HarnessDockerfile is empty")
	}
	text := string(body)
	if !strings.Contains(text, "FROM ocaml/opam:debian-13-ocaml-5.4") {
		t.Errorf("harness Dockerfile does not use the Debian 13 released-compiler base:\n%s", text)
	}
	if !strings.Contains(text, "opam install -y dune ppxlib") {
		t.Errorf("harness Dockerfile does not install the build toolchain:\n%s", text)
	}
}

func TestHarnessImageContext(t *testing.T) {
	reader, err := HarnessImageContext()
	if err != nil {
		t.Fatalf("HarnessImageContext: %v", err)
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read context: %v", err)
	}
	names := tarNames(t, raw)
	if !names[HarnessDockerfilePath] {
		t.Errorf("context is missing %q; members: %v", HarnessDockerfilePath, names)
	}
	if len(names) != 1 {
		t.Errorf("context should hold only the Dockerfile, got members: %v", names)
	}
}
