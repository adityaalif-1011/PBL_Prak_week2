package main

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
)

// ============================================================================
// STUDENT HANDLER STRUCT
// ============================================================================

type StudentHandler struct {
	repo repository.StudentRepository
}

func NewStudentHandler(repo repository.StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func reqContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

func parseListQuery(c *fiber.Ctx) model.ListQuery {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	q := model.ListQuery{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search", ""),
		Sort:   c.Query("sort", ""),
		Order:  c.Query("order", "asc"),
	}

	if isActive := c.Query("is_active"); isActive != "" {
		val := isActive == "true"
		q.IsActive = &val
	}

	if minGrade := c.Query("min_grade"); minGrade != "" {
		q.MinGrade, _ = strconv.Atoi(minGrade)
	}
	if maxGrade := c.Query("max_grade"); maxGrade != "" {
		q.MaxGrade, _ = strconv.Atoi(maxGrade)
	}

	return q
}

func parseID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func translateError(err error) (int, string) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return fiber.StatusNotFound, "data not found"
	case errors.Is(err, repository.ErrDuplicate):
		return fiber.StatusConflict, "NIM already exists"
	default:
		return fiber.StatusInternalServerError, "internal server error"
	}
}

// ============================================================================
// HANDLER METHODS
// ============================================================================

// List handles GET /api/v1/students
func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := reqContext(c)
	defer cancel()

	q := parseListQuery(c)

	students, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		status, msg := translateError(err)
		return c.Status(status).JSON(model.ResponseEnvelope{
			Success: false,
			Message: msg,
		})
	}

	totalPages := 0
	if q.Limit > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}

	return c.Status(fiber.StatusOK).JSON(model.ResponseEnvelope{
		Success: true,
		Message: "students retrieved successfully",
		Data:    students,
		Meta: &model.MetaData{
			Page:       q.Page,
			Limit:      q.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// Get handles GET /api/v1/students/:id
func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := reqContext(c)
	defer cancel()

	id, ok := parseID(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(model.ResponseEnvelope{
			Success: false,
			Message: "invalid ID format",
		})
	}

	student, err := h.repo.FindByID(ctx, id)
	if err != nil {
		status, msg := translateError(err)
		return c.Status(status).JSON(model.ResponseEnvelope{
			Success: false,
			Message: msg,
		})
	}

	return c.Status(fiber.StatusOK).JSON(model.ResponseEnvelope{
		Success: true,
		Message: "student found",
		Data:    student,
	})
}

// Create handles POST /api/v1/students
func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := reqContext(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ResponseEnvelope{
			Success: false,
			Message: "invalid JSON body",
		})
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	errs := map[string]string{}
	if req.NIM == "" {
		errs["nim"] = "NIM is required"
	}
	if req.Name == "" {
		errs["name"] = "name is required"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "grade must be between 0 and 100"
	}

	if len(errs) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(model.ResponseEnvelope{
			Success: false,
			Message: "validation failed",
			Errors:  errs,
		})
	}

	student := model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	}

	created, err := h.repo.Create(ctx, student)
	if err != nil {
		status, msg := translateError(err)
		return c.Status(status).JSON(model.ResponseEnvelope{
			Success: false,
			Message: msg,
		})
	}

	c.Set("Location", "/api/v1/students/"+strconv.Itoa(created.ID))

	return c.Status(fiber.StatusCreated).JSON(model.ResponseEnvelope{
		Success: true,
		Message: "student created successfully",
		Data:    created,
	})
}

// Replace handles PUT /api/v1/students/:id
func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := reqContext(c)
	defer cancel()

	id, ok := parseID(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(model.ResponseEnvelope{
			Success: false,
			Message: "invalid ID format",
		})
	}

	var req model.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ResponseEnvelope{
			Success: false,
			Message: "invalid JSON body",
		})
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	errs := map[string]string{}
	if req.NIM == "" {
		errs["nim"] = "NIM is required"
	}
	if req.Name == "" {
		errs["name"] = "name is required"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "grade must be between 0 and 100"
	}

	if len(errs) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(model.ResponseEnvelope{
			Success: false,
			Message: "validation failed",
			Errors:  errs,
		})
	}

	student := model.Student{
		ID:       id,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	}

	updated, err := h.repo.Update(ctx, student)
	if err != nil {
		status, msg := translateError(err)
		return c.Status(status).JSON(model.ResponseEnvelope{
			Success: false,
			Message: msg,
		})
	}

	return c.Status(fiber.StatusOK).JSON(model.ResponseEnvelope{
		Success: true,
		Message: "student updated successfully",
		Data:    updated,
	})
}

// Patch handles PATCH /api/v1/students/:id
func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := reqContext(c)
	defer cancel()

	id, ok := parseID(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(model.ResponseEnvelope{
			Success: false,
			Message: "invalid ID format",
		})
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ResponseEnvelope{
			Success: false,
			Message: "invalid JSON body",
		})
	}

	existing, err := h.repo.FindByID(ctx, id)
	if err != nil {
		status, msg := translateError(err)
		return c.Status(status).JSON(model.ResponseEnvelope{
			Success: false,
			Message: msg,
		})
	}

	if req.NIM != nil {
		existing.NIM = *req.NIM
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(model.ResponseEnvelope{
				Success: false,
				Message: "validation failed",
				Errors:  map[string]string{"grade": "grade must be between 0 and 100"},
			})
		}
		existing.Grade = *req.Grade
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	updated, err := h.repo.Update(ctx, existing)
	if err != nil {
		status, msg := translateError(err)
		return c.Status(status).JSON(model.ResponseEnvelope{
			Success: false,
			Message: msg,
		})
	}

	return c.Status(fiber.StatusOK).JSON(model.ResponseEnvelope{
		Success: true,
		Message: "student patched successfully",
		Data:    updated,
	})
}

// Delete handles DELETE /api/v1/students/:id
func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := reqContext(c)
	defer cancel()

	id, ok := parseID(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(model.ResponseEnvelope{
			Success: false,
			Message: "invalid ID format",
		})
	}

	err := h.repo.Delete(ctx, id)
	if err != nil {
		status, msg := translateError(err)
		return c.Status(status).JSON(model.ResponseEnvelope{
			Success: false,
			Message: msg,
		})
	}

	return c.Status(fiber.StatusNoContent).JSON(nil)
}
