package supervisor

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeTar(t *testing.T, headers []*tar.Header, bodies [][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for i, hdr := range headers {
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar header %d: %v", i, err)
		}
		if len(bodies[i]) > 0 {
			if _, err := tw.Write(bodies[i]); err != nil {
				t.Fatalf("tar body %d: %v", i, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	return buf.Bytes()
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestExtractTarMembers(t *testing.T) {
	dest := t.TempDir()
	body := []byte("payload")
	data := writeTar(t, []*tar.Header{
		{Name: "dir/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "dir/file.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))},
		{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "dir/file.txt"},
		{Name: "fifo", Typeflag: tar.TypeFifo, Mode: 0o644},
	}, [][]byte{nil, body, nil, nil})

	if err := extractTar(bytes.NewReader(data), dest, ""); err != nil {
		t.Fatalf("extractTar: %v", err)
	}
	if got := readFile(t, filepath.Join(dest, "dir", "file.txt")); got != "payload" {
		t.Errorf("file = %q", got)
	}
	target, err := os.Readlink(filepath.Join(dest, "link"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != "dir/file.txt" {
		t.Errorf("link target = %q", target)
	}
	if _, err := os.Lstat(filepath.Join(dest, "fifo")); !os.IsNotExist(err) {
		t.Error("fifo member was not skipped")
	}
}

func TestExtractTarRejectsEscape(t *testing.T) {
	dest := t.TempDir()
	body := []byte("pwned")
	data := writeTar(t, []*tar.Header{
		{Name: "../evil.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))},
	}, [][]byte{body})

	if err := extractTar(bytes.NewReader(data), dest, ""); err == nil {
		t.Fatal("extractTar accepted a ../ member")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "evil.txt")); !os.IsNotExist(err) {
		t.Fatal("extractTar wrote outside dest")
	}
}

func TestExtractTarRefusesSymlinkedParent(t *testing.T) {
	dest := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dest, "sneak")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	body := []byte("x")
	data := writeTar(t, []*tar.Header{
		{Name: "sneak/file.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))},
	}, [][]byte{body})

	if err := extractTar(bytes.NewReader(data), dest, ""); err == nil {
		t.Fatal("extractTar wrote through a symlinked parent")
	}
	if _, err := os.Stat(filepath.Join(outside, "file.txt")); !os.IsNotExist(err) {
		t.Fatal("extractTar escaped through a symlink")
	}
}

func TestExtractTarReplacesSymlink(t *testing.T) {
	dest := t.TempDir()
	outside := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(outside, []byte("untouched"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	body := []byte("new")
	data := writeTar(t, []*tar.Header{
		{Name: "link", Typeflag: tar.TypeSymlink, Linkname: outside},
		{Name: "link", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))},
	}, [][]byte{nil, body})

	if err := extractTar(bytes.NewReader(data), dest, ""); err != nil {
		t.Fatalf("extractTar: %v", err)
	}
	info, err := os.Lstat(filepath.Join(dest, "link"))
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("link mode = %v, want a regular file", info.Mode())
	}
	if got := readFile(t, filepath.Join(dest, "link")); got != "new" {
		t.Errorf("link content = %q, want new", got)
	}
	if got := readFile(t, outside); got != "untouched" {
		t.Errorf("outside = %q, want untouched", got)
	}
}
