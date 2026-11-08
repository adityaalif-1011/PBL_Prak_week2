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
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid student ID")
	}

	// Cek student ada
	_, err = s.studentRepo.FindByID(c.UserContext(), studentID)
	if err != nil {
		return helper.NewError(fiber.StatusNotFound, "not_found", "student not found")
	}

	list, err := s.repo.GetByStudentID(c.UserContext(), studentID)
	if err != nil {
		return helper.InternalError(err)
	}

	return helper.Success(c, fiber.StatusOK, "achievements retrieved", list)
}

// GET /api/v1/achievements/:id
func (s *AchievementService) Get(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID")
	}

	a, err := s.repo.FindByID(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, repository.ErrAchievementNotFound) {
			return helper.NewError(fiber.StatusNotFound, "not_found", "achievement not found")
		}
		return helper.InternalError(err)
	}

	return helper.Success(c, fiber.StatusOK, "achievement found", a)
}

// POST /api/v1/achievements
func (s *AchievementService) Create(c *fiber.Ctx) error {
	var req model.CreateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	// Cek student ada
	_, err := s.studentRepo.FindByID(c.UserContext(), req.StudentID)
	if err != nil {
		return helper.NewError(fiber.StatusNotFound, "not_found", "student not found")
	}

	created, err := s.repo.Create(c.UserContext(), req)
	if err != nil {
		return helper.InternalError(err)
	}

	return helper.Created(c, "achievement created", created, "/api/v1/achievements/"+strconv.Itoa(created.ID))
}

// PUT /api/v1/achievements/:id
func (s *AchievementService) Update(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID")
	}

	var req model.UpdateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	updated, err := s.repo.Update(c.UserContext(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrAchievementNotFound) {
			return helper.NewError(fiber.StatusNotFound, "not_found", "achievement not found")
		}
		return helper.InternalError(err)
	}

	return helper.Success(c, fiber.StatusOK, "achievement updated", updated)
}

// PATCH /api/v1/achievements/:id
func (s *AchievementService) Patch(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID")
	}

	var req model.PatchAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	current, err := s.repo.FindByID(c.UserContext(), id)
	if err != nil {
		return helper.NewError(fiber.StatusNotFound, "not_found", "achievement not found")
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
		return helper.InternalError(err)
	}

	return helper.Success(c, fiber.StatusOK, "achievement patched", updated)
}

// DELETE /api/v1/achievements/:id
func (s *AchievementService) Delete(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID")
	}

	if err := s.repo.Delete(c.UserContext(), id); err != nil {
		if errors.Is(err, repository.ErrAchievementNotFound) {
			return helper.NewError(fiber.StatusNotFound, "not_found", "achievement not found")
		}
		return helper.InternalError(err)
	}

	return helper.NoContent(c)
}
