package models

import "testing"

func TestRoleValidity(t *testing.T) {
	for _, role := range []Role{RoleUser, RoleModerator, RoleAdmin} {
		if !role.Valid() {
			t.Errorf("%q should be a valid role", role)
		}
	}
	if Role("superadmin").Valid() {
		t.Error("superadmin should no longer be a valid role")
	}
}

func TestCanManageAssessment(t *testing.T) {
	authorID := int64(7)
	otherAuthorID := int64(8)
	tests := []struct {
		name     string
		user     User
		authorID *int64
		want     bool
	}{
		{name: "admin manages any assessment", user: User{Role: RoleAdmin}, want: true},
		{name: "moderator manages own assessment", user: User{ID: authorID, Role: RoleModerator}, authorID: &authorID, want: true},
		{name: "moderator cannot manage another assessment", user: User{ID: authorID, Role: RoleModerator}, authorID: &otherAuthorID, want: false},
		{name: "moderator cannot manage unowned assessment", user: User{ID: authorID, Role: RoleModerator}, want: false},
		{name: "user cannot manage own assessment", user: User{ID: authorID, Role: RoleUser}, authorID: &authorID, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.CanManageAssessment(tt.authorID); got != tt.want {
				t.Errorf("CanManageAssessment() = %v, want %v", got, tt.want)
			}
		})
	}
}
