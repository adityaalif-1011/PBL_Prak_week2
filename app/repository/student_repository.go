package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/model"
)

// Sentinel errors 
var (
	ErrNotFound  = errors.New("data not found")
	ErrDuplicate = errors.New("duplicate data")
)

// StudentRepository adalah KONTAK penyimpanan data student
// Tidak ada satu pun kata "SQL" atau "postgres" di sini
type StudentRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	Create(ctx context.Context, s model.Student) (model.Student, error)
	Update(ctx context.Context, s model.Student) (model.Student, error)
	Delete(ctx context.Context, id int) error
}

// Implementasi PostgreSQL
type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

// Constructor mengembalikan interface, bukan struct konkret
func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

// Whitelist kolom untuk ORDER BY 
var sortColumns = map[string]string{
	"id":         "id",
	"nim":        "nim",
	"name":       "name",
	"grade":      "grade",
	"is_active":  "is_active",
	"created_at": "created_at",
}

// buildFilter menyusun WHERE clause dengan parameter
// Semua nilai dari klien jadi parameter ($1, $2, ...), bukan disambung ke SQL
func buildFilter(q model.ListQuery) (string, []interface{}) {
	where := "WHERE 1=1"
	args := []interface{}{}

	if q.Search != "" {
		where += fmt.Sprintf(" AND name ILIKE $%d", len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}

	if q.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", len(args)+1)
		args = append(args, *q.IsActive)
	}

	if q.MinGrade > 0 {
		where += fmt.Sprintf(" AND grade >= $%d", len(args)+1)
		args = append(args, q.MinGrade)
	}

	if q.MaxGrade > 0 {
		where += fmt.Sprintf(" AND grade <= $%d", len(args)+1)
		args = append(args, q.MaxGrade)
	}

	return where, args
}

// ============================================================================
// IMPLEMENTASI METHOD
// ============================================================================

// FindAll returns paginated students with total count
func (r *studentPostgresRepository) FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error) {
	where, args := buildFilter(q)

	// 1. Hitung total untuk meta
	var total int
	countQuery := "SELECT COUNT(*) FROM students " + where
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count students: %w", err)
	}

	// 2. Validasi sort column - whitelist
	sortCol := "id"
	if col, ok := sortColumns[q.Sort]; ok {
		sortCol = col
	}

	order := "ASC"
	if strings.ToLower(q.Order) == "desc" {
		order = "DESC"
	}

	// 3. Query dengan LIMIT dan OFFSET
	query := fmt.Sprintf(
		"SELECT id, nim, name, grade, is_active, created_at FROM students %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sortCol, order, len(args)+1, len(args)+2,
	)

	args = append(args, q.Limit, q.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query students: %w", err)
	}
	defer rows.Close()

	students := []model.Student{}
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan row: %w", err)
		}
		students = append(students, s)
	}

	return students, total, nil
}

// FindByID returns one student by ID
func (r *studentPostgresRepository) FindByID(ctx context.Context, id int) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		"SELECT id, nim, name, grade, is_active, created_at FROM students WHERE id = $1",
		id,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("get student: %w", err)
	}
	return s, nil
}

// Create inserts new student and returns with generated ID and created_at
func (r *studentPostgresRepository) Create(ctx context.Context, s model.Student) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		"INSERT INTO students (nim, name, grade, is_active) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		s.NIM, s.Name, s.Grade, s.IsActive,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("create student: %w", err)
	}
	return s, nil
}

// Update replaces all fields of a student
func (r *studentPostgresRepository) Update(ctx context.Context, s model.Student) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		"UPDATE students SET nim = $1, name = $2, grade = $3, is_active = $4 WHERE id = $5 RETURNING created_at",
		s.NIM, s.Name, s.Grade, s.IsActive, s.ID,
	).Scan(&s.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("update student: %w", err)
	}
	return s, nil
}

// Delete removes a student by ID
func (r *studentPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM students WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete student: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// isUniqueViolation checks if error is PostgreSQL unique violation (23505)
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}