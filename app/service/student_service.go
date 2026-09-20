package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

// ============================================================================
// STUDENT SERVICE STRUCT
// ============================================================================

type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentService(repo repository.StudentRepository, perms *helper.PermissionSet) *StudentService {
	return &StudentService{repo: repo, perms: perms}
}

// ============================================================================
// HELPER
// ============================================================================

// ownerOf mengekstrak owner_id dari student (bisa nil).
func ownerOf(s model.Student) int {
	if s.OwnerID == nil {
		return 0
	}
	return *s.OwnerID
}

// ============================================================================
// HANDLER METHODS
// ============================================================================

// List handles GET /api/v1/students
func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseListQuery(c)

	students, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		status, msg := translateError(err)
		return helper.Fail(c, status, msg)
	}

	totalPages := CountTotalPages(total, q.Limit)

	return helper.SuccessList(c, "students retrieved successfully", students, &model.MetaData{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// Get handles GET /api/v1/students/:id
func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "invalid ID format")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		status, msg := translateError(err)
		return helper.Fail(c, status, msg)
	}

	// Ownership check
	if !CanAccessStudent(current, ownerOf(student), s.perms, "student:read:any") {
		return helper.Fail(c, fiber.StatusForbidden, "tidak berhak mengakses data student ini")
	}

	return helper.Success(c, fiber.StatusOK, "student found", student)
}

// Create handles POST /api/v1/students
func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "invalid JSON body")
	}

	if errs := ValidateCreate(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	student := model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	}

	// owner_id diisi dari identitas pemanggil, BUKAN dari body
	created, err := s.repo.Create(ctx, student, current.UserID)
	if err != nil {
		status, msg := translateError(err)
		return helper.Fail(c, status, msg)
	}

	return helper.Created(c, "student created successfully", created, "/api/v1/students/"+strconv.Itoa(created.ID))
}

// Replace handles PUT /api/v1/students/:id
func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "invalid ID format")
	}

	// Ambil data dulu untuk tahu owner-nya
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		status, msg := translateError(err)
		return helper.Fail(c, status, msg)
	}

	// Ownership check
	if !CanAccessStudent(current, ownerOf(existing), s.perms, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "tidak berhak mengubah data student ini")
	}

	var req model.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "invalid JSON body")
	}

	if errs := ValidateReplace(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	student := model.Student{
		ID:       id,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	}

	updated, err := s.repo.Update(ctx, student)
	if err != nil {
		status, msg := translateError(err)
		return helper.Fail(c, status, msg)
	}

	return helper.Success(c, fiber.StatusOK, "student updated successfully", updated)
}

// Patch handles PATCH /api/v1/students/:id
func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "invalid ID format")
	}

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		status, msg := translateError(err)
		return helper.Fail(c, status, msg)
	}

	// Ownership check
	if !CanAccessStudent(current, ownerOf(existing), s.perms, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "tidak berhak mengubah data student ini")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "invalid JSON body")
	}

	if IsEmptyPatch(req) {
		return helper.Fail(c, fiber.StatusBadRequest, "no fields to update")
	}

	updated, errs := ApplyPatch(existing, req)
	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	result, err := s.repo.Update(ctx, updated)
	if err != nil {
		status, msg := translateError(err)
		return helper.Fail(c, status, msg)
	}

	return helper.Success(c, fiber.StatusOK, "student patched successfully", result)
}

// Delete handles DELETE /api/v1/students/:id
func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.Fail(c, fiber.StatusBadRequest, "invalid ID format")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		status, msg := translateError(err)
		return helper.Fail(c, status, msg)
	}

	return helper.NoContent(c)
}

// ============================================================================
// ERROR TRANSLATION
// ============================================================================

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
