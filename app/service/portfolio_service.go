package service

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

// PortfolioService menangani portofolio mandiri student (MongoDB).
type PortfolioService struct {
	portfolios *repository.PortfolioRepository
	students   repository.StudentRepository
}

func NewPortfolioService(
	portfolios *repository.PortfolioRepository,
	students repository.StudentRepository,
) *PortfolioService {
	return &PortfolioService{portfolios: portfolios, students: students}
}

// Get - GET /api/v1/students/:id/portfolio.
func (s *PortfolioService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid student ID")
	}
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		return helper.NewError(fiber.StatusNotFound, "not_found", "student not found")
	}

	p, err := s.portfolios.FindByStudentID(ctx, studentID)
	if err != nil {
		if err == repository.ErrPortfolioNotFound {
			// Belum ada portofolio: kembalikan kerangka kosong.
			return helper.Success(c, fiber.StatusOK, "portfolio not found", model.Portfolio{
				StudentID: studentID,
				Projects:  []model.Project{},
				Skills:    []string{},
			})
		}
		return helper.InternalError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portfolio found", p)
}

// Upsert - PUT /api/v1/students/:id/portfolio.
func (s *PortfolioService) Upsert(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid student ID")
	}
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		return helper.NewError(fiber.StatusNotFound, "not_found", "student not found")
	}

	var req model.UpsertPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	p, err := s.portfolios.Upsert(ctx, studentID, req)
	if err != nil {
		return helper.InternalError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portfolio saved", p)
}

// AddProject - POST /api/v1/students/:id/portfolio/projects.
func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid student ID")
	}
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		return helper.NewError(fiber.StatusNotFound, "not_found", "student not found")
	}

	var req model.AddProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	p, err := s.portfolios.AddProject(ctx, studentID, req)
	if err != nil {
		switch err {
		case repository.ErrPortfolioNotFound:
			return helper.NewError(fiber.StatusNotFound, "not_found", "portfolio not found")
		case repository.ErrProjectLimit:
			return helper.NewError(fiber.StatusConflict, "project_limit", "project limit reached (max 20)")
		default:
			return helper.InternalError(err)
		}
	}
	return helper.Success(c, fiber.StatusOK, "project added", p)
}

// DeleteProject - DELETE /api/v1/students/:id/portfolio/projects/:projectId.
func (s *PortfolioService) DeleteProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid student ID")
	}
	projectID := c.Params("projectId")
	if projectID == "" {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid project ID")
	}
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		return helper.NewError(fiber.StatusNotFound, "not_found", "student not found")
	}

	p, err := s.portfolios.DeleteProject(ctx, studentID, projectID)
	if err != nil {
		switch err {
		case repository.ErrPortfolioNotFound:
			return helper.NewError(fiber.StatusNotFound, "not_found", "portfolio not found")
		case repository.ErrProjectNotFound:
			return helper.NewError(fiber.StatusNotFound, "not_found", "project not found")
		default:
			return helper.InternalError(err)
		}
	}
	return helper.Success(c, fiber.StatusOK, "project deleted", p)
}

// TopTechnologies - GET /api/v1/students/:id/portfolio/technologies.
func (s *PortfolioService) TopTechnologies(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid student ID")
	}
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		return helper.NewError(fiber.StatusNotFound, "not_found", "student not found")
	}

	stats, err := s.portfolios.TopTechnologies(ctx, studentID)
	if err != nil {
		return helper.InternalError(err)
	}
	return helper.SuccessList(c, "technologies retrieved", stats, nil)
}

// GlobalStats - GET /api/v1/portfolios/stats/technologies.
func (s *PortfolioService) GlobalStats(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	stats, err := s.portfolios.TopTechnologiesAll(ctx)
	if err != nil {
		return helper.InternalError(err)
	}
	return helper.SuccessList(c, "technology stats retrieved", stats, nil)
}

// Location helper agar import strconv terpakai bila diperlukan nanti.
var _ = strconv.Itoa
