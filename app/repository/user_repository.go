package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/model"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUserDuplicate = errors.New("user already exists")
)

// UserRepository adalah KONTRAK penyimpanan data user.
// Berbentuk interface agar bisa dipalsukan (fake) di unit test.
type UserRepository interface {
	// Create menambah user baru.
	Create(ctx context.Context, u model.User) (model.User, error)
	// FindByUsername mencari user untuk login (case-insensitive).
	FindByUsername(ctx context.Context, username string) (model.User, error)
	// FindAfterCursor mengambil halaman user setelah cursor (keyset pagination).
	FindAfterCursor(ctx context.Context, q model.UserQuery) ([]model.User, error)
	// FindByID mengambil satu user berdasarkan id.
	FindByID(ctx context.Context, id int) (model.User, error)
	// WithTx mengembalikan salinan repository yang query-nya berjalan di dalam tx.
	WithTx(tx DBTX) UserRepository
}

// Implementasi PostgreSQL.
type userPostgresRepository struct {
	db DBTX
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userPostgresRepository{db: pool}
}

// WithTx mengembalikan salinan repository yang query-nya berjalan di dalam tx.
func (r *userPostgresRepository) WithTx(tx DBTX) UserRepository {
	return &userPostgresRepository{db: tx}
}

// Create - tambah user baru
func (r *userPostgresRepository) Create(ctx context.Context, u model.User) (model.User, error) {
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (username, email, password, role, is_active) 
		 VALUES ($1, $2, $3, $4, $5) 
		 RETURNING id, created_at`,
		u.Username, u.Email, u.Password, u.Role, u.IsActive,
	).Scan(&u.ID, &u.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) { // ← dari student_repository.go
			return model.User{}, ErrUserDuplicate
		}
		return model.User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// FindByUsername - untuk login
func (r *userPostgresRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx,
		`SELECT id, username, email, password, role, is_active, created_at 
		 FROM users WHERE LOWER(username) = LOWER($1)`,
		username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("find user: %w", err)
	}
	return u, nil
}

// FindAfterCursor - daftar user setelah cursor (keyset pagination, tanpa OFFSET).
// Query memakai WHERE (created_at, id) < ($1, $2) dan LIMIT q.Limit + 1
// agar service tahu apakah masih ada halaman berikutnya.
func (r *userPostgresRepository) FindAfterCursor(ctx context.Context, q model.UserQuery) ([]model.User, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	where := "WHERE (created_at, id) < ($1, $2)"
	args := []any{q.CursorAt, q.CursorID}

	if q.Search != "" {
		args = append(args, "%"+q.Search+"%")
		where += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)", len(args), len(args))
	}

	args = append(args, limit+1)
	query := fmt.Sprintf(
		`SELECT id, username, email, role, is_active, created_at
		 FROM users
		 %s
		 ORDER BY created_at DESC, id DESC
		 LIMIT $%d`,
		where, len(args),
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find users after cursor: %w", err)
	}
	defer rows.Close()

	users := []model.User{}
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}

// FindByID - untuk endpoint /me
func (r *userPostgresRepository) FindByID(ctx context.Context, id int) (model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx,
		`SELECT id, username, email, password, role, is_active, created_at 
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("find user: %w", err)
	}
	return u, nil
}
