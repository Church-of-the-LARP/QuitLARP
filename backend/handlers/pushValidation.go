package handlers

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"backend/assessment"
	"backend/database"
	"backend/repos"
)

const (
	// maxPushValidationBody caps how much of the callback body is read; the
	// hook only ever sends a handful of short lines.
	maxPushValidationBody = 64 << 10
	// zeroSHA is the all-zero object id git uses for "no commit".
	zeroSHA = "0000000000000000000000000000000000000000"
	// assessmentSyncTimeout bounds one detached repository -> database sync.
	assessmentSyncTimeout = 2 * time.Minute
)

// PushValidationHandler answers the callback posted by the git update hook
// before a push to main is allowed to move the ref. It runs in the context of
// the pushing client's request and decides the push's fate.
func (h *Handlers) PushValidationHandler() http.Handler {
	return http.HandlerFunc(h.pushValidation)
}

func (h *Handlers) pushValidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeText(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.pushSecret == "" ||
		subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Push-Validation")), []byte(h.pushSecret)) != 1 {
		writeText(w, http.StatusUnauthorized, "missing or invalid push validation credentials")
		return
	}

	start := time.Now()
	fields := readPushFields(r)

	repository := fields["repository"]
	ref := fields["ref"]
	oldSHA := fields["old"]
	newSHA := fields["new"]
	quarantine := fields["quarantine"]
	if repository == "" || ref == "" || newSHA == "" {
		writeText(w, http.StatusBadRequest, "malformed push validation request")
		return
	}

	// The hook already filters these; accepting again keeps the endpoint
	// harmless if it is called with something else.
	if ref != "refs/heads/main" {
		writeText(w, http.StatusOK, "")
		return
	}
	id, ok := parseAssessmentRepoID(repository)
	if !ok {
		writeText(w, http.StatusOK, "")
		return
	}

	if newSHA == zeroSHA {
		rejectPush(w, repository, newSHA, "main cannot be deleted", "main cannot be deleted")
		return
	}

	if _, err := database.GetAssessmentAuthorID(r.Context(), h.db, id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			rejectPush(w, repository, newSHA, "no assessment backs this repository", "no assessment backs this repository")
			return
		}
		log.Printf("push validation: assessment lookup for %s: %v", repository, err)
		rejectPush(w, repository, newSHA, "could not verify that main only moves forward", "could not verify that main only moves forward")
		return
	}

	dir, err := h.gitRepoDir(repository)
	if err != nil {
		log.Printf("push validation: repository dir for %s: %v", repository, err)
		rejectPush(w, repository, newSHA, "could not verify that main only moves forward", "could not verify that main only moves forward")
		return
	}

	if oldSHA != zeroSHA {
		ok, err := fastForwardOK(dir, oldSHA, newSHA, quarantine)
		switch {
		case err != nil:
			log.Printf("push validation: merge-base %s %s in %s: %v", oldSHA, newSHA, dir, err)
			rejectPush(w, repository, newSHA, "could not verify that main only moves forward", "could not verify that main only moves forward")
			return
		case !ok:
			rejectPush(w, repository, newSHA,
				"main must only move forward: rewriting history is not allowed",
				"main must only move forward: rewriting history is not allowed")
			return
		}
	}

	tmp, err := os.MkdirTemp("", "assessment-push-*")
	if err != nil {
		log.Printf("push validation: temp dir: %v", err)
		rejectPush(w, repository, newSHA, "could not read the pushed snapshot: "+err.Error(), "could not read the pushed snapshot: "+err.Error())
		return
	}
	defer os.RemoveAll(tmp)

	ctx := r.Context()
	if err := assessment.Materialize(ctx, dir, newSHA, quarantine, tmp); err != nil {
		log.Printf("push validation: materialize %s %s: %v", repository, newSHA, err)
		body := "could not read the pushed snapshot: " + err.Error()
		rejectPush(w, repository, newSHA, err.Error(), body)
		return
	}

	content, violations := assessment.Load(tmp)
	if len(violations) > 0 {
		var b strings.Builder
		b.WriteString("push to main was rejected:")
		for _, v := range violations {
			b.WriteString("\n- ")
			b.WriteString(v)
		}
		rejectPush(w, repository, newSHA, strings.Join(violations, "; "), b.String())
		return
	}

	failures := make([]string, 0, len(content.Chapters))
	var body strings.Builder
	for _, ch := range content.Chapters {
		report, err := h.generator.Generate(ctx, filepath.Join(tmp, filepath.FromSlash(ch.Dir)), ch.Specs)
		if err == nil {
			continue
		}
		failures = append(failures, fmt.Sprintf("chapter %s: %v", ch.Dir, err))
		if body.Len() == 0 {
			body.WriteString("binding generation failed:")
		}
		body.WriteString("\nchapter ")
		body.WriteString(ch.Dir)
		body.WriteString(": ")
		body.WriteString(err.Error())
		if report != "" {
			body.WriteString("\n")
			body.WriteString(strings.TrimRight(report, "\n"))
		}
	}
	if len(failures) > 0 {
		rejectPush(w, repository, newSHA, strings.Join(failures, "; "), body.String())
		return
	}

	log.Printf("push validation accepted assessment %d %s: %d chapters in %s",
		id, newSHA, len(content.Chapters), time.Since(start).Round(time.Millisecond))
	writeText(w, http.StatusOK, "")
}

// readPushFields parses the key=value body the hook posts. Unknown keys and
// blank lines are ignored so adding a field to the hook never breaks an older
// backend.
func readPushFields(r *http.Request) map[string]string {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxPushValidationBody))
	if err != nil {
		return map[string]string{}
	}
	fields := make(map[string]string, 5)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimRight(line, "\r")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "repository", "ref", "old", "new", "quarantine":
			fields[key] = value
		}
	}
	return fields
}

// fastForwardOK reports whether newSHA descends from oldSHA. quarantine is the
// hook's GIT_QUARANTINE_PATH, where the push's objects live until the ref
// moves, so the check runs against the repository plus that directory.
func fastForwardOK(dir, oldSHA, newSHA, quarantine string) (bool, error) {
	cmd := exec.Command("git", "--git-dir="+dir, "merge-base", "--is-ancestor", oldSHA, newSHA)
	cmd.Env = os.Environ()
	if quarantine != "" {
		cmd.Env = append(cmd.Env, "GIT_ALTERNATE_OBJECT_DIRECTORIES="+quarantine)
	}
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// rejectPush refuses a push: the client sees the body as remote lines and the
// operator gets one log line naming the repository, the new sha and why.
func rejectPush(w http.ResponseWriter, repoID, newSHA, reason, body string) {
	log.Printf("push validation rejected %s %s: %s", repoID, newSHA, reason)
	writeText(w, http.StatusUnprocessableEntity, body)
}

// writeText answers with a plain-text body; the git client prints it verbatim.
func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	if body != "" {
		_, _ = io.WriteString(w, body)
	}
}

// syncAssessmentFromRepo reconciles the database with what main now contains
// after a push. It runs detached from the push request, so every failure is
// logged and dropped instead of being returned to a client.
func (h *Handlers) syncAssessmentFromRepo(ctx context.Context, repoID string) {
	id, ok := parseAssessmentRepoID(repoID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, assessmentSyncTimeout)
	defer cancel()

	dir, err := h.gitRepoDir(repoID)
	if err != nil {
		log.Printf("assessment sync %s: %v", repoID, err)
		return
	}
	repo, err := repos.OpenDir(dir)
	if err != nil {
		log.Printf("assessment sync %s: %v", repoID, err)
		return
	}
	sha, err := repo.Revision("main")
	if err != nil {
		if !errors.Is(err, repos.ErrRefNotFound) {
			log.Printf("assessment sync %s: %v", repoID, err)
		}
		return
	}
	current, err := database.GetAssessmentMainCommit(ctx, h.db, id)
	if err != nil {
		log.Printf("assessment sync %s: %v", repoID, err)
		return
	}
	if current == sha {
		return
	}

	tmp, err := os.MkdirTemp("", "assessment-sync-*")
	if err != nil {
		log.Printf("assessment sync %s: %v", repoID, err)
		return
	}
	defer os.RemoveAll(tmp)

	if err := assessment.Materialize(ctx, dir, sha, "", tmp); err != nil {
		log.Printf("assessment sync %s: %v", repoID, err)
		return
	}
	content, violations := assessment.Load(tmp)
	if len(violations) > 0 {
		log.Printf("assessment sync %s: main violates the layout, not syncing: %s", repoID, strings.Join(violations, "; "))
		return
	}
	if err := database.SyncAssessmentContent(ctx, h.db, id, content, sha); err != nil {
		log.Printf("assessment sync %s: %v", repoID, err)
		return
	}
	log.Printf("assessment sync %s: synced %s (%d chapters)", repoID, sha, len(content.Chapters))
}
