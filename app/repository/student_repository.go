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
	ErrDuplicate = errors.New("duplicate data")
	ErrNotFound  = errors.New("student not found")
)

// StudentRepository adalah KONTRAK penyimpanan data student
type StudentRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	Create(ctx context.Context, s model.Student, ownerID int) (model.Student, error)
	Update(ctx context.Context, s model.Student) (model.Student, error)
	Delete(ctx context.Context, id int) error
}

// Implementasi PostgreSQL
type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

var sortColumns = map[string]string{
	"id":         "id",
	"nim":        "nim",
	"name":       "name",
	"grade":      "grade",
	"is_active":  "is_active",
	"created_at": "created_at",
}

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

// FindAll
func (r *studentPostgresRepository) FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error) {
	where, args := buildFilter(q)

	var total int
	countQuery := "SELECT COUNT(*) FROM students " + where
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count students: %w", err)
	}

	sortCol := "id"
	if col, ok := sortColumns[q.Sort]; ok {
		sortCol = col
	}

	order := "ASC"
	if strings.ToLower(q.Order) == "desc" {
		order = "DESC"
	}

	query := fmt.Sprintf(
		"SELECT id, nim, name, grade, is_active, owner_id, created_at FROM students %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
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
		if err := rows.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan row: %w", err)
		}
		students = append(students, s)
	}
	return students, total, nil
}

// FindByID
func (r *studentPostgresRepository) FindByID(ctx context.Context, id int) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		"SELECT id, nim, name, grade, is_active, owner_id, created_at FROM students WHERE id = $1",
		id,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("get student: %w", err)
	}
	return s, nil
}

// Create — ownerID diisi dari parameter (dari identitas pemanggil)
func (r *studentPostgresRepository) Create(ctx context.Context, s model.Student, ownerID int) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		"INSERT INTO students (nim, name, grade, is_active, owner_id) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at",
		s.NIM, s.Name, s.Grade, s.IsActive, ownerID,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("create student: %w", err)
	}
	s.OwnerID = &ownerID
	return s, nil
}

// Update
func (r *studentPostgresRepository) Update(ctx context.Context, s model.Student) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		"UPDATE students SET nim = $1, name = $2, grade = $3, is_active = $4 WHERE id = $5 RETURNING owner_id, created_at",
		s.NIM, s.Name, s.Grade, s.IsActive, s.ID,
	).Scan(&s.OwnerID, &s.CreatedAt)

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

// Delete
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
