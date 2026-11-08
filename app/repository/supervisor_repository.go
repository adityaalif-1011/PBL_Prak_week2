package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"api-students/app/model"
)

var (
	ErrSupervisorNotFound  = errors.New("supervisor not found")
	ErrSupervisionNotFound = errors.New("supervision not found")
	ErrAlreadySupervised   = errors.New("student already has a supervisor")
	ErrSupervisorFull      = errors.New("supervisor has reached max students")
)

// SupervisorRepository menangani supervisors & supervisions.
type SupervisorRepository struct {
	db DBTX
}

func NewSupervisorRepository(db DBTX) *SupervisorRepository {
	return &SupervisorRepository{db: db}
}

func (r *SupervisorRepository) WithTx(tx DBTX) *SupervisorRepository {
	return &SupervisorRepository{db: tx}
}

// ============================================================
// SUPERVISORS
// ============================================================

func (r *SupervisorRepository) Create(ctx context.Context, req model.CreateSupervisorRequest) (model.Supervisor, error) {
	max := req.MaxStudents
	if max <= 0 {
		max = 10
	}
	var s model.Supervisor
	err := r.db.QueryRow(ctx,
		`INSERT INTO supervisors (name, max_students) VALUES ($1, $2)
		 RETURNING id, name, max_students, created_at`,
		req.Name, max,
	).Scan(&s.ID, &s.Name, &s.MaxStudents, &s.CreatedAt)
	if err != nil {
		return model.Supervisor{}, fmt.Errorf("create supervisor: %w", err)
	}
	return s, nil
}

func (r *SupervisorRepository) FindByID(ctx context.Context, id int) (model.Supervisor, error) {
	var s model.Supervisor
	err := r.db.QueryRow(ctx,
		`SELECT id, name, max_students, created_at FROM supervisors WHERE id = $1`, id,
	).Scan(&s.ID, &s.Name, &s.MaxStudents, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrSupervisorNotFound
		}
		return model.Supervisor{}, fmt.Errorf("find supervisor: %w", err)
	}
	return s, nil
}

// LockSupervisor - kunci baris supervisor (SELECT ... FOR UPDATE) supaya
// pengecekan kapasitas aman dari race condition.
func (r *SupervisorRepository) LockSupervisor(ctx context.Context, id int) (model.Supervisor, error) {
	var s model.Supervisor
	err := r.db.QueryRow(ctx,
		`SELECT id, name, max_students, created_at FROM supervisors WHERE id = $1 FOR UPDATE`, id,
	).Scan(&s.ID, &s.Name, &s.MaxStudents, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrSupervisorNotFound
		}
		return model.Supervisor{}, fmt.Errorf("lock supervisor: %w", err)
	}
	return s, nil
}

func (r *SupervisorRepository) Patch(ctx context.Context, id int, req model.PatchSupervisorRequest) (model.Supervisor, error) {
	s, err := r.FindByID(ctx, id)
	if err != nil {
		return model.Supervisor{}, err
	}
	if req.Name != nil {
		s.Name = *req.Name
	}
	if req.MaxStudents != nil {
		s.MaxStudents = *req.MaxStudents
	}
	var updated model.Supervisor
	err = r.db.QueryRow(ctx,
		`UPDATE supervisors SET name = $2, max_students = $3 WHERE id = $1
		 RETURNING id, name, max_students, created_at`,
		id, s.Name, s.MaxStudents,
	).Scan(&updated.ID, &updated.Name, &updated.MaxStudents, &updated.CreatedAt)
	if err != nil {
		return model.Supervisor{}, fmt.Errorf("patch supervisor: %w", err)
	}
	return updated, nil
}

// CountAssigned - banyak student yang sedang dibimbing supervisor.
func (r *SupervisorRepository) CountAssigned(ctx context.Context, supervisorID int) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM supervisions WHERE supervisor_id = $1`, supervisorID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count assigned: %w", err)
	}
	return n, nil
}

// ============================================================
// SUPERVISIONS
// ============================================================

func (r *SupervisorRepository) Assign(ctx context.Context, supervisorID, studentID int) (model.Supervision, error) {
	// Pastikan student belum punya pembimbing (student_id UNIQUE).
	var exists int
	if err := r.db.QueryRow(ctx,
		`SELECT 1 FROM supervisions WHERE student_id = $1`, studentID).Scan(&exists); err == nil {
		return model.Supervision{}, ErrAlreadySupervised
	}

	var sup model.Supervision
	err := r.db.QueryRow(ctx,
		`INSERT INTO supervisions (supervisor_id, student_id) VALUES ($1, $2)
		 RETURNING id, supervisor_id, student_id, created_at`,
		supervisorID, studentID,
	).Scan(&sup.ID, &sup.SupervisorID, &sup.StudentID, &sup.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Supervision{}, ErrAlreadySupervised
		}
		return model.Supervision{}, fmt.Errorf("assign supervision: %w", err)
	}
	return sup, nil
}

// Switch - pindahkan student ke pembimbing lain dalam satu UPDATE.
func (r *SupervisorRepository) Switch(ctx context.Context, studentID, newSupervisorID int) (model.Supervision, error) {
	var sup model.Supervision
	err := r.db.QueryRow(ctx,
		`UPDATE supervisions SET supervisor_id = $2 WHERE student_id = $1
		 RETURNING id, supervisor_id, student_id, created_at`,
		studentID, newSupervisorID,
	).Scan(&sup.ID, &sup.SupervisorID, &sup.StudentID, &sup.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervision{}, ErrSupervisionNotFound
		}
		return model.Supervision{}, fmt.Errorf("switch supervision: %w", err)
	}
	return sup, nil
}

// FindByStudent - ambil supervision milik satu student.
func (r *SupervisorRepository) FindByStudent(ctx context.Context, studentID int) (model.Supervision, error) {
	var sup model.Supervision
	err := r.db.QueryRow(ctx,
		`SELECT id, supervisor_id, student_id, created_at FROM supervisions WHERE student_id = $1`,
		studentID,
	).Scan(&sup.ID, &sup.SupervisorID, &sup.StudentID, &sup.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervision{}, ErrSupervisionNotFound
		}
		return model.Supervision{}, fmt.Errorf("find supervision: %w", err)
	}
	return sup, nil
}
