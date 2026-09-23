package assessment

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Materialize writes commit sha of the bare repository at repoDir into destDir,
// which is created when missing.
//
// quarantine, when non-empty, is the object directory of an in-flight receive
// (git sets GIT_QUARANTINE_PATH for the update hook).
func Materialize(ctx context.Context, repoDir, sha, quarantine, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("materialize %s: create %s: %w", sha, destDir, err)
	}

	cmd := exec.CommandContext(ctx, "git", "--git-dir="+repoDir, "archive", "--format=tar", sha)
	cmd.Env = quarantineEnv(os.Environ(), quarantine)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("materialize %s: pipe git archive stdout: %w", sha, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("materialize %s: start git archive: %w", sha, err)
	}
	if err := extractTar(stdout, destDir); err != nil {
		// Drain the pipe so git cannot block writing into it while we wait.
		_, _ = io.Copy(io.Discard, stdout)
		_ = cmd.Wait()
		return fmt.Errorf("materialize %s: extract into %s: %w", sha, destDir, err)
	}
	if err := cmd.Wait(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return fmt.Errorf("materialize %s: git archive: %w", sha, err)
		}
		return fmt.Errorf("materialize %s: git archive: %w: %s", sha, err, message)
	}
	return nil
}

// quarantineEnv returns env with GIT_ALTERNATE_OBJECT_DIRECTORIES pointing at
// quarantine first.
//
// This is unorthodox: git runs receive-pack in a quarantine and publishes the
// object directory it is writing there as GIT_QUARANTINE_PATH, so a freshly
// pushed commit lives only in that directory until the push is accepted. The
// update hook runs before acceptance, so `git archive` can only resolve the sha
// when it is told to look in the quarantine through the alternate object
// directories mechanism. Any pre-existing value is kept after the quarantine so
// the repository's own alternates still resolve.
func quarantineEnv(env []string, quarantine string) []string {
	if quarantine == "" {
		return env
	}
	const key = "GIT_ALTERNATE_OBJECT_DIRECTORIES="
	out := make([]string, 0, len(env)+1)
	var existing string
	for _, kv := range env {
		if strings.HasPrefix(kv, key) {
			existing = strings.TrimPrefix(kv, key)
			continue
		}
		out = append(out, kv)
	}
	value := quarantine
	if existing != "" {
		value = quarantine + string(os.PathListSeparator) + existing
	}
	return append(out, key+value)
}

// extractTar unpacks a tar stream into destDir. It is split out from Materialize
// so the path safety rules can be tested directly.
func extractTar(r io.Reader, destDir string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := extractMember(destDir, hdr, tr); err != nil {
			return err
		}
	}
}

// extractMember writes one tar member under destDir.
//
// The safety rules matter because a pushed tar is attacker controlled: a member
// may not escape destDir (absolute paths, ".." traversal), may not be written
// through a symlinked parent directory (the symlink would redirect the write
// outside destDir), and symlinks are created as symlinks but never followed.
func extractMember(destDir string, hdr *tar.Header, r io.Reader) error {
	if hdr.Name == "" {
		return nil
	}
	target := filepath.Join(destDir, filepath.FromSlash(hdr.Name))
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
// writes through a link that an earlier member created. Removing the link
// itself leaves its target untouched.
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
