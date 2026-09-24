package handlers

import (
	"bytes"
	"context"
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

// pushHook is installed as hooks/update in every assessment repository. It
// asks the backend to validate a push before the main ref moves; updates of
// any other ref pass straight through. The client sees rejections as remote
// lines, exactly like a server-side pre-receive hook.
const pushHook = `#!/bin/sh
# Installed by the backend. Updates of main are validated before the ref moves;
# other refs pass through. Rejections reach the client as remote lines.
ref="$1"
old="$2"
new="$3"

[ "$ref" = "refs/heads/main" ] || exit 0

if [ "$new" = "0000000000000000000000000000000000000000" ]; then
	echo "main cannot be deleted" >&2
	exit 1
fi

if [ -z "$PUSH_VALIDATION_URL" ] || [ -z "$PUSH_VALIDATION_SECRET" ]; then
	echo "push validation is not configured; refusing to update main" >&2
	exit 1
fi

result=$(
	printf 'repository=%s\nref=%s\nold=%s\nnew=%s\nquarantine=%s\n' \
		"$PUSH_REPOSITORY_ID" "$ref" "$old" "$new" "$GIT_QUARANTINE_PATH" |
	curl -sS -m 1800 -w 'HTTP_STATUS:%{http_code}' \
		-H "X-Push-Validation: $PUSH_VALIDATION_SECRET" \
		--data-binary @- "$PUSH_VALIDATION_URL"
) || {
	echo "push validation is unavailable; try again later" >&2
	exit 1
}

status=${result##*HTTP_STATUS:}
body=${result%HTTP_STATUS:*}

if [ "$status" != "200" ]; then
	printf '%s\n' "$body" >&2
	exit 1
fi

exit 0
`

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
	u, ok := h.gitUser(w, r)
	if !ok {
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

	if id, ok := parseAssessmentRepoID(repoID); ok {
		if err := h.authorizeAssessmentRepo(r.Context(), u, id); err != nil {
			writeAssessmentRepoFailure(w, err)
			return
		}
	}
	if assessmentID, solverID, ok := parseSolutionRepoID(repoID); ok {
		if err := h.authorizeSolutionRepo(r.Context(), u, assessmentID, solverID, service); err != nil {
			writeSolutionRepoFailure(w, err)
			return
		}
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

	u, ok := h.gitUser(w, r)
	if !ok {
		return
	}

	if id, ok := parseAssessmentRepoID(repoID); ok {
		if err := h.authorizeAssessmentRepo(r.Context(), u, id); err != nil {
			writeAssessmentRepoFailure(w, err)
			return
		}
	}
	if assessmentID, solverID, ok := parseSolutionRepoID(repoID); ok {
		if err := h.authorizeSolutionRepo(r.Context(), u, assessmentID, solverID, service); err != nil {
			writeSolutionRepoFailure(w, err)
			return
		}
	}

	dir, err := h.gitRepoDir(repoID)
	if err != nil {
		writeGitFailure(w, err)
		return
	}

	cmd := exec.CommandContext(r.Context(), "git", gitCommandName(service), "--stateless-rpc", dir)
	cmd.Env = append(os.Environ(), gitProtocolEnv(r)...)
	if service == gitReceivePack {
		cmd.Env = append(cmd.Env,
			"PUSH_VALIDATION_URL=http://127.0.0.1:"+h.cfg.Port+"/internal/push-validation",
			"PUSH_VALIDATION_SECRET="+h.pushSecret,
			"PUSH_REPOSITORY_ID="+repoID,
		)
	}
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
		go h.syncAssessmentFromRepo(context.Background(), repoID)
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

	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
			return "", fmt.Errorf("create repos dir: %w", err)
		}
		// Bare: no working tree to get in the way of pushes. HEAD starts on
		// "main" to match what modern tooling expects.
		out, err := exec.Command("git", "init", "--bare", "--quiet", "--initial-branch=main", dir).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git init %s: %w: %s", dir, err, strings.TrimSpace(string(out)))
		}
	}

	if _, ok := parseAssessmentRepoID(id); ok {
		if err := ensurePushHook(dir); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// ensurePushHook keeps hooks/update of an assessment repository identical to
// pushHook. It rewrites the file whenever the content differs, so upgrading
// the constant takes effect on the next push.
func ensurePushHook(dir string) error {
	path := filepath.Join(dir, "hooks", "update")
	if current, err := os.ReadFile(path); err == nil && string(current) == pushHook {
		return os.Chmod(path, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("install push hook in %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(pushHook), 0o755); err != nil {
		return fmt.Errorf("install push hook in %s: %w", dir, err)
	}
	return nil
}

// parseAssessmentRepoID recognises the repository name that backs an
// assessment ("assessments/assessment-<id>", with an optional ".git" suffix)
// and returns the assessment id it encodes.
func parseAssessmentRepoID(repoID string) (int64, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSuffix(repoID, ".git"), "assessments/assessment-")
	if !ok {
		return 0, false
	}
	return parsePositiveID(rest)
}

// parseSolutionRepoID recognises the repository name that holds one user's
// solution for an assessment ("solutions/assessment-<id>/user-<uid>", with an
// optional ".git" suffix) and returns the ids it encodes.
func parseSolutionRepoID(repoID string) (int64, int64, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSuffix(repoID, ".git"), "solutions/assessment-")
	if !ok {
		return 0, 0, false
	}
	assessmentText, userText, ok := strings.Cut(rest, "/user-")
	if !ok {
		return 0, 0, false
	}
	assessmentID, ok := parsePositiveID(assessmentText)
	if !ok {
		return 0, 0, false
	}
	userID, ok := parsePositiveID(userText)
	if !ok {
		return 0, 0, false
	}
	return assessmentID, userID, true
}

// parsePositiveID parses a decimal repository id, rejecting empty values and
// leading zeros so one repository cannot be addressed by several names.
func parsePositiveID(text string) (int64, bool) {
	if text == "" || (len(text) > 1 && text[0] == '0') {
		return 0, false
	}
	for _, c := range text {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	id, err := strconv.ParseInt(text, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// errNoAssessmentRepo is the sentinel for a repository with no assessment
// behind it.
var errNoAssessmentRepo = errors.New("no assessment backs this repository")

// errAssessmentRepoForbidden is the sentinel for a caller who may not use an
// assessment repository.
var errAssessmentRepoForbidden = errors.New("forbidden")

// authorizeAssessmentRepo allows the author of an assessment, or an
// admin/superadmin, to use its repository. Everything else, including an
// assessment id with no row behind it, is refused so a push cannot lazily
// create a repository for an assessment that does not exist.
func (h *Handlers) authorizeAssessmentRepo(ctx context.Context, u models.User, id int64) error {
	authorID, err := database.GetAssessmentAuthorID(ctx, h.db, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return errNoAssessmentRepo
		}
		return fmt.Errorf("assessment author lookup: %w", err)
	}
	if (authorID != nil && *authorID == u.ID) ||
		u.Role == models.RoleAdmin || u.Role == models.RoleSuperadmin {
		return nil
	}
	return errAssessmentRepoForbidden
}

// writeAssessmentRepoFailure turns an authorization failure into a response.
func writeAssessmentRepoFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errNoAssessmentRepo):
		writeProblem(w, http.StatusNotFound, "no assessment backs this repository")
	case errors.Is(err, errAssessmentRepoForbidden):
		writeProblem(w, http.StatusForbidden, "you do not have permission to use this repository")
	default:
		log.Printf("assessment repository authorization: %v", err)
		writeProblem(w, http.StatusInternalServerError, "git service unavailable")
	}
}

// errSolutionReadOnly is the sentinel for a write to a solution repository.
var errSolutionReadOnly = errors.New("solution repositories are read-only")

// errNoSolutionRepo is the sentinel for a solution that has no row behind it.
var errNoSolutionRepo = errors.New("no solution exists for this user and assessment")

// authorizeSolutionRepo allows the solver, the assessment's author and
// admins to read a solution; public solutions may be read by any signed-in
// user. Writes are always refused: only the backend commits to solutions.
func (h *Handlers) authorizeSolutionRepo(ctx context.Context, u models.User, assessmentID, solverID int64, service string) error {
	if service == gitReceivePack {
		return errSolutionReadOnly
	}
	if u.ID == solverID {
		return nil
	}
	authorID, err := database.GetAssessmentAuthorID(ctx, h.db, assessmentID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return errNoAssessmentRepo
		}
		return fmt.Errorf("solution assessment lookup: %w", err)
	}
	if (authorID != nil && *authorID == u.ID) ||
		u.Role == models.RoleAdmin || u.Role == models.RoleSuperadmin {
		return nil
	}
	solution, err := database.GetSolution(ctx, h.db, assessmentID, solverID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return errNoSolutionRepo
		}
		return fmt.Errorf("solution lookup: %w", err)
	}
	if solution.Public {
		return nil
	}
	return errAssessmentRepoForbidden
}

// writeSolutionRepoFailure turns a solution authorization failure into a
// response.
func writeSolutionRepoFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errSolutionReadOnly):
		writeProblem(w, http.StatusForbidden, "solution repositories are read-only")
	case errors.Is(err, errNoAssessmentRepo):
		writeProblem(w, http.StatusNotFound, "no assessment backs this repository")
	case errors.Is(err, errNoSolutionRepo):
		writeProblem(w, http.StatusNotFound, "no solution exists for this user and assessment")
	case errors.Is(err, errAssessmentRepoForbidden):
		writeProblem(w, http.StatusForbidden, "you do not have permission to read this solution")
	default:
		log.Printf("solution repository authorization: %v", err)
		writeProblem(w, http.StatusInternalServerError, "git service unavailable")
	}
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
