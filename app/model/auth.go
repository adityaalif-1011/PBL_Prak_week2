package model

import "time"

// RegisterRequest - request untuk daftar
type RegisterRequest struct {
	Username string `json:"username" validate:"required,nospace,username"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,strongpassword"`
	// TIDAK ADA field Role di sini!
	// Kalau ada, siapa pun bisa daftar sebagai admin (mass assignment)
}

// LoginRequest - request untuk login
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest - request untuk refresh token
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// TokenPair - pasangan access token dan refresh token
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // detik
}

// User - user yang bisa login
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // tidak pernah keluar sebagai JSON
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// RefreshToken - row di tabel refresh_tokens
type RefreshToken struct {
	ID        int64
	UserID    int
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// AuthUser - identitas yang dibawa access token
type AuthUser struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
