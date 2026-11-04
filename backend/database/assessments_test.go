package database

import (
	"reflect"
	"testing"
)

func TestAssessmentWhereFiltersAuthorID(t *testing.T) {
	authorID := int64(42)
	where, args := assessmentWhere(AssessmentFilter{AuthorID: &authorID})

	if where != " WHERE a.author_id = $1" {
		t.Fatalf("where = %q, want author ownership filter", where)
	}
	if !reflect.DeepEqual(args, []any{authorID}) {
		t.Fatalf("args = %#v, want author id %d", args, authorID)
	}
}
