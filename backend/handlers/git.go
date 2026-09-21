package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"backend/auth"
	"backend/database"
	"backend/middleware"
	"backend/models"
)

const (
	gitUploadPack  = "git-upload-pack"
	gitReceivePack = "git-receive-pack"
)

var repoPathPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}(/[A-Za-z0-9][A-Za-z0-9._-]{0,63})*$`)

var errInvalidRepo = errors.New("invalid repository name")

func (h *Handlers) GitHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /git/{repo...}", h.gitInfoRefs)
	mux.HandleFunc("POST /git/{repo...}", h.gitRPC)
	return middleware.Authenticate(h.tokens, mux)
}

func (h *Handlers) gitUser(w http.ResponseWriter, r *http.Request) (models.User, bool) {
	if u, err := middleware.CurrentUser(r.Context(), h.db); err == nil {
		return u, true
	} else if !errors.Is(err, middleware.ErrUnauthenticated) {
		writeGitFailure(w, err)
		return models.User{}, false
	}

	if ident, password, ok := r.BasicAuth(); ok && password != "" {
		row, err := database.GetUserWithPassword(r.Context(), h.db, ident)
		if errors.Is(err, database.ErrNotFound) {
			row, err = database.GetUserWithPasswordByUsername(r.Context(), h.db, ident)
		}
		if err == nil && row.PasswordHash != nil && auth.CheckPassword(*row.PasswordHash, password) {
			return row.User, true
		}
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
	writeProblem(w, http.StatusUnauthorized, "authentication required — use your username (or email) and password")
	return models.User{}, false
}

// gitInfoRefs answers GET .../info/refs: the ref advertisement that starts
// every fetch and push.
func (h *Handlers) gitInfoRefs(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.gitUser(w, r); !ok {
		return
	}

	repoID, ok := strings.CutSuffix(r.PathValue("repo"), "/info/refs")
	if !ok {
		http.NotFound(w, r)
		return
	}

	service := r.URL.Query().Get("service")
	if service != gitUploadPack && service != gitReceivePack {
		http.Error(w, "only the smart HTTP protocol is supported", http.StatusForbidden)
		return
	}

	dir, err := h.gitRepoDir(repoID)
	if err != nil {
		writeGitFailure(w, err)
		return
	}

	cmd := exec.CommandContext(r.Context(), "git", gitCommandName(service), "--stateless-rpc", "--advertise-refs", dir)
	cmd.Env = append(os.Environ(), gitProtocolEnv(r)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		writeGitFailure(w, fmt.Errorf("git %s --advertise-refs in %s: %w: %s",
			service, dir, err, strings.TrimSpace(stderr.String())))
		return
	}

	body := append(gitServiceHeader(service), stdout.Bytes()...)
	w.Header().Set("Content-Type", "application/x-"+service+"-advertisement")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = w.Write(body)
}

// gitRPC proxies a POST body to `git <service> --stateless-rpc` and streams
// the result back (including the progress sideband for pushes).
func (h *Handlers) gitRPC(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("repo")
	var service, repoID string
	switch {
	case strings.HasSuffix(path, "/"+gitUploadPack):
		service, repoID = gitUploadPack, strings.TrimSuffix(path, "/"+gitUploadPack)
	case strings.HasSuffix(path, "/"+gitReceivePack):
		service, repoID = gitReceivePack, strings.TrimSuffix(path, "/"+gitReceivePack)
	default:
		http.NotFound(w, r)
		return
	}

	if _, ok := h.gitUser(w, r); !ok {
		return
	}

	dir, err := h.gitRepoDir(repoID)
	if err != nil {
		writeGitFailure(w, err)
		return
	}

	cmd := exec.CommandContext(r.Context(), "git", gitCommandName(service), "--stateless-rpc", dir)
	cmd.Env = append(os.Environ(), gitProtocolEnv(r)...)
	cmd.Stdin = r.Body
	defer r.Body.Close()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeGitFailure(w, fmt.Errorf("git %s in %s: %w", service, dir, err))
		return
	}
	if err := cmd.Start(); err != nil {
		writeGitFailure(w, fmt.Errorf("git %s in %s: %w", service, dir, err))
		return
	}

	// git consumes the request body as it produces output; we forward
	// that output as it arrives.
	w.Header().Set("Content-Type", "application/x-"+service+"-result")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(flushWriter{w}, stdout)

	if err := cmd.Wait(); err != nil {
		log.Printf("git %s in %s failed: %v: %s", service, dir, err, strings.TrimSpace(stderr.String()))
		return
	}
	if service == gitReceivePack {
		gitAlignHead(dir)
	}
}

// gitRepoDir resolves the bare repository for repoID and creates it on first
// use. The ".git" suffix git clients put in URLs is accepted and normalized
// away, so /git/demo and /git/demo.git are the same repository. Nested names
// such as assessments/assessment-1 map to nested directories.
func (h *Handlers) gitRepoDir(repoID string) (string, error) {
	id := strings.TrimSuffix(repoID, ".git")
	if len(id) > 255 || !repoPathPattern.MatchString(id) {
		return "", fmt.Errorf("%w: %q", errInvalidRepo, repoID)
	}
	dir := filepath.Join(h.cfg.GitReposDir, filepath.FromSlash(id+".git"))

	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err == nil {
		return dir, nil
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", fmt.Errorf("create repos dir: %w", err)
	}
	// Bare: no working tree to get in the way of pushes. HEAD starts on
	// "main" to match what modern tooling expects.
	out, err := exec.Command("git", "init", "--bare", "--quiet", "--initial-branch=main", dir).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git init %s: %w: %s", dir, err, strings.TrimSpace(string(out)))
	}
	return dir, nil
}

// gitAlignHead points HEAD at the repository's only branch when HEAD does not
// resolve to a commit yet. A first push that used, say, "master" instead of
// the initial "main" would otherwise leave clones unable to check anything
// out. Repos with a healthy HEAD (or with zero/multiple branches) are left
// alone.
func gitAlignHead(dir string) {
	if err := exec.Command("git", "--git-dir", dir, "rev-parse", "--verify", "--quiet", "HEAD").Run(); err == nil {
		return
	}
	out, err := exec.Command("git", "--git-dir", dir, "for-each-ref", "--format=%(refname:short)", "refs/heads").Output()
	if err != nil {
		return
	}
	branches := strings.Fields(string(out))
	if len(branches) != 1 {
		return
	}
	if err := exec.Command("git", "--git-dir", dir, "symbolic-ref", "HEAD", "refs/heads/"+branches[0]).Run(); err != nil {
		log.Printf("git: could not point HEAD of %s at %s: %v", dir, branches[0], err)
	}
}

// gitCommandName maps an HTTP service name ("git-upload-pack") to the git
// subcommand that runs it ("upload-pack"). git dispatches subcommands by
// prepending "git-", so passing the full service name would send it looking
// for "git-git-upload-pack".
func gitCommandName(service string) string {
	return strings.TrimPrefix(service, "git-")
}

// gitServiceHeader is the pkt-line announcing the service, which the smart
// protocol prefixes to the ref advertisement ("001e# service=git-upload-pack\n0000").
func gitServiceHeader(service string) []byte {
	line := "# service=" + service + "\n"
	return fmt.Appendf(nil, "%04x%s0000", len(line)+4, line)
}

// gitProtocolEnv forwards the protocol version the client asked for (e.g.
// "version=2") so git answers in a protocol that client speaks.
func gitProtocolEnv(r *http.Request) []string {
	if p := r.Header.Get("Git-Protocol"); p == "version=2" || p == "version=1" {
		return []string{"GIT_PROTOCOL=" + p}
	}
	return nil
}

// flushWriter streams every write straight to the client, so a clone sees
// pack data and progress as git produces it instead of at the end.
type flushWriter struct{ w http.ResponseWriter }

func (f flushWriter) Write(p []byte) (int, error) {
	n, err := f.w.Write(p)
	if fl, ok := f.w.(http.Flusher); ok {
		fl.Flush()
	}
	return n, err
}

// writeGitFailure turns a handler-side failure into a response. A bad repo
// name is the only client-fixable case and gets its own status; everything
// else is a server-side problem and is logged for the operator.
func writeGitFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalidRepo):
		writeProblem(w, http.StatusBadRequest, "invalid repository name: path segments use letters, digits, '.', '-' or '_'")
	case errors.Is(err, exec.ErrNotFound):
		log.Printf("git: the git binary is not installed — the local git server needs it (see backend/Dockerfile)")
		writeProblem(w, http.StatusInternalServerError, "git service unavailable")
	default:
		log.Printf("git: %v", err)
		writeProblem(w, http.StatusInternalServerError, "git service unavailable")
	}
}
