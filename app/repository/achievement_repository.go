package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/model"
)

var ErrAchievementNotFound = errors.New("achievement not found")

type AchievementRepository struct {
	pool *pgxpool.Pool
}

func NewAchievementRepository(pool *pgxpool.Pool) *AchievementRepository {
	return &AchievementRepository{pool: pool}
}

// GetByStudentID - ambil semua prestasi dari 1 student
func (r *AchievementRepository) GetByStudentID(ctx context.Context, studentID int) ([]model.Achievement, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, student_id, name, champion_level, created_at FROM achievements WHERE student_id = $1 ORDER BY id",
		studentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Achievement
	for rows.Next() {
		var a model.Achievement
		rows.Scan(&a.ID, &a.StudentID, &a.Name, &a.ChampionLevel, &a.CreatedAt)
		list = append(list, a)
	}
	return list, nil
}

// FindByID - ambil 1 prestasi
func (r *AchievementRepository) FindByID(ctx context.Context, id int) (model.Achievement, error) {
	var a model.Achievement
	err := r.pool.QueryRow(ctx,
		"SELECT id, student_id, name, champion_level, created_at FROM achievements WHERE id = $1",
		id,
	).Scan(&a.ID, &a.StudentID, &a.Name, &a.ChampionLevel, &a.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Achievement{}, ErrAchievementNotFound
		}
		return model.Achievement{}, err
	}
	return a, nil
}

// Create - tambah prestasi
func (r *AchievementRepository) Create(ctx context.Context, req model.CreateAchievementRequest) (model.Achievement, error) {
	var a model.Achievement
	err := r.pool.QueryRow(ctx,
		"INSERT INTO achievements (student_id, name, champion_level) VALUES ($1, $2, $3) RETURNING id, created_at",
		req.StudentID, req.Name, req.ChampionLevel,
	).Scan(&a.ID, &a.CreatedAt)

	if err != nil {
		return model.Achievement{}, err
	}

	a.StudentID = req.StudentID
	a.Name = req.Name
	a.ChampionLevel = req.ChampionLevel
	return a, nil
}

// Update - ubah prestasi
func (r *AchievementRepository) Update(ctx context.Context, id int, req model.UpdateAchievementRequest) (model.Achievement, error) {
	var a model.Achievement
	err := r.pool.QueryRow(ctx,
		"UPDATE achievements SET name = $1, champion_level = $2 WHERE id = $3 RETURNING id, student_id, name, champion_level, created_at",
		req.Name, req.ChampionLevel, id,
	).Scan(&a.ID, &a.StudentID, &a.Name, &a.ChampionLevel, &a.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Achievement{}, ErrAchievementNotFound
		}
		return model.Achievement{}, err
	}
	return a, nil
}

// Delete - hapus prestasi
func (r *AchievementRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM achievements WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAchievementNotFound
	}
	return nil
}
