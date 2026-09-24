package supervisor

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractTar unpacks a tar stream into destDir, which is created when missing.
// stripPrefix, when non-empty, is the member name prefix to strip: the daemon
// serves an archive of a directory whose members are rooted at that
// directory's own name, so the caller passes the base name of the source path.
//
// The stream comes from the sandboxed container and is therefore untrusted: a
// member may not escape destDir (absolute paths, ".." traversal), may not be
// written through a symlinked parent directory (the symlink would redirect the
// write outside destDir), and symlinks are created as symlinks but never
// followed. It mirrors the push-time extractor in the assessment package
// without importing it, so the two can evolve independently.
func extractTar(r io.Reader, destDir, stripPrefix string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := extractMember(destDir, stripPrefix, hdr, tr); err != nil {
			return err
		}
	}
}

// extractMember writes one tar member under destDir.
func extractMember(destDir, stripPrefix string, hdr *tar.Header, r io.Reader) error {
	name := strings.TrimPrefix(hdr.Name, "./")
	if stripPrefix != "" {
		switch {
		case name == stripPrefix:
			// The source directory itself; destDir already stands for it.
			return nil
		case strings.HasPrefix(name, stripPrefix+"/"):
			name = strings.TrimPrefix(name, stripPrefix+"/")
		default:
			return fmt.Errorf("tar member %q is outside %q", hdr.Name, stripPrefix)
		}
	}
	if name == "" {
		return nil
	}
	target := filepath.Join(destDir, filepath.FromSlash(name))
	if !withinDir(destDir, target) {
		return fmt.Errorf("tar member %q escapes the destination", hdr.Name)
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		if err := removeSymlink(target); err != nil {
			return err
		}
		return os.MkdirAll(target, memberMode(hdr.Mode, 0o755))
	case tar.TypeReg:
		if err := mkdirForFile(destDir, target); err != nil {
			return err
		}
		if err := removeSymlink(target); err != nil {
			return err
		}
		return writeFile(target, r, memberMode(hdr.Mode, 0o644))
	case tar.TypeSymlink:
		if err := mkdirForFile(destDir, target); err != nil {
			return err
		}
		if err := removeSymlink(target); err != nil {
			return err
		}
		return os.Symlink(hdr.Linkname, target)
	default:
		// Hard links, devices, fifos and other irregular members are skipped.
		return nil
	}
}

// removeSymlink removes path when it is an existing symlink, so a member never
// writes through a link an earlier member created. Removing the link itself
// leaves its target untouched.
func removeSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(path)
	}
	return nil
}

// withinDir reports whether target, after cleaning, stays inside dir.
func withinDir(dir, target string) bool {
	dir = filepath.Clean(dir)
	rel, err := filepath.Rel(dir, filepath.Clean(target))
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// mkdirForFile creates the parent of target, refusing to walk through a symlink.
func mkdirForFile(destDir, target string) error {
	parent := filepath.Dir(target)
	if err := refuseSymlinkParents(destDir, parent); err != nil {
		return err
	}
	return os.MkdirAll(parent, 0o755)
}

// refuseSymlinkParents rejects a destination whose existing parents include a
// symlink, which would make the write land outside destDir.
func refuseSymlinkParents(destDir, dir string) error {
	rel, err := filepath.Rel(filepath.Clean(destDir), filepath.Clean(dir))
	if err != nil {
		return err
	}
	current := filepath.Clean(destDir)
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write through symlinked directory %s", current)
		}
	}
	return nil
}

// writeFile creates target with the given permissions.
func writeFile(target string, r io.Reader, perm os.FileMode) error {
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// memberMode masks the permission bits of a tar member to 0o777 so setuid,
// setgid and sticky bits are dropped, falling back to a sane default when the
// member carries no permission bits at all.
func memberMode(mode int64, fallback os.FileMode) os.FileMode {
	if perm := os.FileMode(mode) & 0o777; perm != 0 {
		return perm
	}
	return fallback
}
