package repos

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
)

// Revision resolves ref to the commit sha it points at.
func (r *Repo) Revision(ref string) (string, error) {
	hash, err := r.repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrRefNotFound, ref)
	}
	return hash.String(), nil
}
