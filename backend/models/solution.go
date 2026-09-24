package models

import (
	"fmt"
	"time"
)

// Solution is one persisted submission: the candidate's file(s) committed
// into a bare repository on the local git server, plus who may read it.
// Public solutions are readable by any signed-in user, everything else by the
// solver, the assessment's author and admins.
type Solution struct {
	ID           int64      `json:"id" db:"id" doc:"Unique database identifier"`
	AssessmentID int64      `json:"assessmentId" db:"assessment_id" doc:"The assessment that was solved"`
	UserID       int64      `json:"userId" db:"user_id" doc:"The user who solved it"`
	Public       bool       `json:"public" db:"is_public" doc:"Whether every signed-in user may read the solution"`
	CommitSHA    string     `json:"commitSha" db:"commit_sha" doc:"Commit the solution was written in"`
	Author       *AuthorRef `json:"author" doc:"The person who solved it"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at" doc:"First submission time"`
	UpdatedAt    time.Time  `json:"updatedAt" db:"updated_at" doc:"Last submission time"`
}

// SolutionRepoID is the path of the bare repository holding one user's
// solution for one assessment.
func SolutionRepoID(assessmentID, userID int64) string {
	return fmt.Sprintf("solutions/assessment-%d/user-%d", assessmentID, userID)
}
