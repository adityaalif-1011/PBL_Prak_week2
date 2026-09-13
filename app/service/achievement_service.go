package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type AchievementService struct {
	repo        *repository.AchievementRepository
	studentRepo repository.StudentRepository
}

func NewAchievementService(repo *repository.AchievementRepository, studentRepo repository.StudentRepository) *AchievementService {
	return &AchievementService{repo: repo, studentRepo: studentRepo}
}

// GET /api/v1/students/:id/achievements
func (s *AchievementService) GetByStudentID(c *fiber.Ctx) error {
	studentID, err := strconv.Atoi(c.Params("id"))
	if err != nil || studentID <= 0 {
		return helper.Fail(c, 400, "invalid student ID")
	}

	// Cek student ada
	_, err = s.studentRepo.FindByID(c.UserContext(), studentID)
	if err != nil {
		return helper.Fail(c, 404, "student not found")
	}

	list, err := s.repo.GetByStudentID(c.UserContext(), studentID)
	if err != nil {
		return helper.Fail(c, 500, "failed to fetch achievements")
	}

	return helper.Success(c, 200, "achievements retrieved", list)
}

// GET /api/v1/achievements/:id
func (s *AchievementService) Get(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, 400, "invalid ID")
	}

	a, err := s.repo.FindByID(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, repository.ErrAchievementNotFound) {
			return helper.Fail(c, 404, "achievement not found")
		}
		return helper.Fail(c, 500, "failed to fetch achievement")
	}

	return helper.Success(c, 200, "achievement found", a)
}

// POST /api/v1/achievements
func (s *AchievementService) Create(c *fiber.Ctx) error {
	var req model.CreateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, 400, "invalid JSON")
	}

	// Validasi
	errs := map[string]string{}
	if req.StudentID <= 0 {
		errs["student_id"] = "student_id is required"
	}
	if req.Name == "" {
		errs["name"] = "name is required"
	}
	if req.ChampionLevel == "" {
		errs["champion_level"] = "champion_level is required"
	}
	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	// Cek student ada
	_, err := s.studentRepo.FindByID(c.UserContext(), req.StudentID)
	if err != nil {
		return helper.Fail(c, 404, "student not found")
	}

	created, err := s.repo.Create(c.UserContext(), req)
	if err != nil {
		return helper.Fail(c, 500, "failed to create achievement")
	}

	return helper.Created(c, "achievement created", created, "/api/v1/achievements/"+strconv.Itoa(created.ID))
}

// PUT /api/v1/achievements/:id
func (s *AchievementService) Update(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, 400, "invalid ID")
	}

	var req model.UpdateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, 400, "invalid JSON")
	}

	errs := map[string]string{}
	if req.Name == "" {
		errs["name"] = "name is required"
	}
	if req.ChampionLevel == "" {
		errs["champion_level"] = "champion_level is required"
	}
	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	updated, err := s.repo.Update(c.UserContext(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrAchievementNotFound) {
			return helper.Fail(c, 404, "achievement not found")
		}
		return helper.Fail(c, 500, "failed to update achievement")
	}

	return helper.Success(c, 200, "achievement updated", updated)
}

// PATCH /api/v1/achievements/:id
func (s *AchievementService) Patch(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, 400, "invalid ID")
	}

	var req model.PatchAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, 400, "invalid JSON")
	}

	current, err := s.repo.FindByID(c.UserContext(), id)
	if err != nil {
		return helper.Fail(c, 404, "achievement not found")
	}

	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.ChampionLevel != nil {
		current.ChampionLevel = *req.ChampionLevel
	}

	updated, err := s.repo.Update(c.UserContext(), id, model.UpdateAchievementRequest{
		Name:          current.Name,
		ChampionLevel: current.ChampionLevel,
	})
	if err != nil {
		return helper.Fail(c, 500, "failed to patch")
	}

	return helper.Success(c, 200, "achievement patched", updated)
}

// DELETE /api/v1/achievements/:id
func (s *AchievementService) Delete(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, 400, "invalid ID")
	}

	err := s.repo.Delete(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, repository.ErrAchievementNotFound) {
			return helper.Fail(c, 404, "achievement not found")
		}
		return helper.Fail(c, 500, "failed to delete")
	}

	return helper.NoContent(c)
}
