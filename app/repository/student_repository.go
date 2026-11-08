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

	// WithTx mengembalikan salinan repository yang query-nya berjalan di dalam tx.
	WithTx(tx DBTX) StudentRepository
	// CreateWithUser membuat user login + student dalam SATU transaksi.
	// Password user sudah di-hash oleh pemanggil (dari NIM).
	CreateWithUser(ctx context.Context, s model.Student, passwordHash string) (model.Student, model.User, error)
}

// Implementasi PostgreSQL
type studentPostgresRepository struct {
	db DBTX
}

func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{db: pool}
}

func (r *studentPostgresRepository) WithTx(tx DBTX) StudentRepository {
	return &studentPostgresRepository{db: tx}
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
	where := "WHERE deleted_at IS NULL"
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

const studentCols = "id, nim, name, grade, is_active, total_sks, batas_sks, owner_id, created_at"

func scanStudent(row pgx.Row) (model.Student, error) {
	var s model.Student
	err := row.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive,
		&s.TotalSKS, &s.BatasSKS, &s.OwnerID, &s.CreatedAt)
	return s, err
}

// FindAll
func (r *studentPostgresRepository) FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error) {
	where, args := buildFilter(q)

	var total int
	countQuery := "SELECT COUNT(*) FROM students " + where
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
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
		"SELECT %s FROM students %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		studentCols, where, sortCol, order, len(args)+1, len(args)+2,
	)

	args = append(args, q.Limit, q.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query students: %w", err)
	}
	defer rows.Close()

	students := []model.Student{}
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive,
			&s.TotalSKS, &s.BatasSKS, &s.OwnerID, &s.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan row: %w", err)
		}
		students = append(students, s)
	}
	return students, total, nil
}

// FindByID
func (r *studentPostgresRepository) FindByID(ctx context.Context, id int) (model.Student, error) {
	s, err := scanStudent(r.db.QueryRow(ctx,
		"SELECT "+studentCols+" FROM students WHERE id = $1 AND deleted_at IS NULL", id))
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
	batas := s.BatasSKS
	if batas <= 0 {
		batas = 24
	}
	err := r.db.QueryRow(ctx,
		"INSERT INTO students (nim, name, grade, is_active, owner_id, batas_sks) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at",
		s.NIM, s.Name, s.Grade, s.IsActive, ownerID, batas,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("create student: %w", err)
	}
	s.OwnerID = &ownerID
	s.BatasSKS = batas
	return s, nil
}

// CreateWithUser - membuat user login + student dalam SATU transaksi.
// username & email user memakai NIM; password sudah di-hash dari NIM.
func (r *studentPostgresRepository) CreateWithUser(ctx context.Context, s model.Student, passwordHash string) (model.Student, model.User, error) {
	batas := s.BatasSKS
	if batas <= 0 {
		batas = 24
	}

	// 1) buat user login (role 'user', aktif)
	var user model.User
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (username, email, password, role, is_active)
		 VALUES ($1, $2, $3, 'user', true)
		 RETURNING id, username, email, role, is_active, created_at`,
		s.NIM, s.NIM+"@students.local", passwordHash,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.IsActive, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, model.User{}, ErrDuplicate
		}
		return model.Student{}, model.User{}, fmt.Errorf("create user for student: %w", err)
	}

	// 2) buat student dengan owner_id = user baru
	ownerID := user.ID
	err = r.db.QueryRow(ctx,
		"INSERT INTO students (nim, name, grade, is_active, owner_id, batas_sks) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at",
		s.NIM, s.Name, s.Grade, s.IsActive, ownerID, batas,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, model.User{}, ErrDuplicate
		}
		return model.Student{}, model.User{}, fmt.Errorf("create student: %w", err)
	}
	s.OwnerID = &ownerID
	s.BatasSKS = batas
	return s, user, nil
}

// Update
func (r *studentPostgresRepository) Update(ctx context.Context, s model.Student) (model.Student, error) {
	err := r.db.QueryRow(ctx,
		"UPDATE students SET nim = $1, name = $2, grade = $3, is_active = $4, batas_sks = $5 WHERE id = $6 AND deleted_at IS NULL RETURNING owner_id, created_at, total_sks",
		s.NIM, s.Name, s.Grade, s.IsActive, s.BatasSKS, s.ID,
	).Scan(&s.OwnerID, &s.CreatedAt, &s.TotalSKS)

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

// Delete - SOFT DELETE: set deleted_at & nonaktifkan, data tidak hilang.
func (r *studentPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx,
		"UPDATE students SET deleted_at = NOW(), is_active = false WHERE id = $1 AND deleted_at IS NULL", id)
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
