// Package models holds the shared domain types used across the backend.
package models

import "time"

// Role is the authorization level of a user.
type Role string

const (
	RoleUser       Role = "user"
	RoleAdmin      Role = "admin"
	RoleSuperadmin Role = "superadmin"
)

// Valid reports whether r is one of the known roles.
func (r Role) Valid() bool {
	switch r {
	case RoleUser, RoleAdmin, RoleSuperadmin:
		return true
	}
	return false
}

// ParseRole converts a string to a Role.
func ParseRole(s string) (Role, bool) {
	r := Role(s)
	return r, r.Valid()
}

// User is the public, serializable representation of an account. Password
// hashes and OAuth identifiers are never exposed outside the data layer.
type User struct {
	ID            int64     `json:"id" db:"id" example:"1" doc:"Unique database identifier"`
	Username      string    `json:"username" db:"username" doc:"Unique display/account name"`
	Email         string    `json:"email" db:"email" doc:"Login email address"`
	EmailVerified bool      `json:"emailVerified" db:"email_verified" doc:"Whether the email address was confirmed"`
	Role          Role      `json:"role" db:"role" enum:"user,admin,superadmin" doc:"Authorization role"`
	GoogleLinked  bool      `json:"googleLinked" db:"google_linked" doc:"Whether a Google account is linked"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at" doc:"Account creation time"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at" doc:"Last modification time"`
}

// UserColumns is the SELECT list used everywhere a User is scanned; it maps
// the google_sub column to the derived googleLinked boolean.
const UserColumns = `id, username, email, email_verified, role,
	google_sub IS NOT NULL AS google_linked, created_at, updated_at`

// UserWithPassword is a User plus its password hash, only used by the data
// layer when checking credentials.
type UserWithPassword struct {
	User
	PasswordHash *string `db:"password_hash"`
}
