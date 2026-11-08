package model

import "time"

// UserQuery - parameter query list user (cursor pagination).
type UserQuery struct {
	Limit    int
	CursorAt time.Time
	CursorID int
	Search   string
}

// CreateUserRequest - request membuat user.
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,nospace,username"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,strongpassword"`
	Role     string `json:"role" validate:"omitempty,oneof=admin user"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// PatchUserRequest - hanya field yang dikirim ikut diubah.
// Semua field pointer dengan tag omitnil agar null != tidak dikirim.
type PatchUserRequest struct {
	Username *string `json:"username,omitnil" validate:"omitempty,nospace,username"`
	Email    *string `json:"email,omitnil" validate:"omitempty,email"`
	Password *string `json:"password,omitnil" validate:"omitempty,strongpassword"`
	Role     *string `json:"role,omitnil" validate:"omitempty,oneof=admin user"`
	IsActive *bool   `json:"is_active,omitnil"`
}

// CursorMeta - meta untuk response berbasis cursor (tanpa offset).
type CursorMeta struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}
