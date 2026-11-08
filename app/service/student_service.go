package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

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
	tx    *repository.TxManager
}

func NewStudentService(repo repository.StudentRepository, perms *helper.PermissionSet, tx *repository.TxManager) *StudentService {
	return &StudentService{repo: repo, perms: perms, tx: tx}
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
		return translateError(err)
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
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "belum terautentikasi")
	}

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID format")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err)
	}

	// Ownership check
	if !CanAccessStudent(current, ownerOf(student), s.perms, "student:read:any") {
		return helper.NewError(fiber.StatusForbidden, "forbidden", "tidak berhak mengakses data student ini")
	}

	return helper.Success(c, fiber.StatusOK, "student found", student)
}

// Create handles POST /api/v1/students
// Membuat user login + student dalam SATU transaksi; password di-hash dari NIM.
func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	student := model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	}

	created, err := s.createStudentWithUser(ctx, student)
	if err != nil {
		return translateError(err)
	}

	return helper.Created(c, "student created successfully", created, "/api/v1/students/"+strconv.Itoa(created.ID))
}

// createStudentWithUser - hash password dari NIM lalu buat user + student
// di dalam satu transaksi (atomic; gagal = tidak ada yang tersimpan).
func (s *StudentService) createStudentWithUser(ctx context.Context, student model.Student) (model.Student, error) {
	// Password awal = NIM, langsung di-hash (bcrypt) sehingga plaintext NIM
	// tidak pernah tersimpan.
	hashed, err := helper.HashPassword(student.NIM)
	if err != nil {
		return model.Student{}, err
	}

	var created model.Student
	err = s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		repo := s.repo.WithTx(tx)
		var u model.User
		created, u, err = repo.CreateWithUser(ctx, student, hashed)
		if err != nil {
			return err
		}
		_ = u
		return nil
	})
	if err != nil {
		return model.Student{}, err
	}
	return created, nil
}

// Replace handles PUT /api/v1/students/:id
func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "belum terautentikasi")
	}

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID format")
	}

	// Ambil data dulu untuk tahu owner-nya
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err)
	}

	// Ownership check
	if !CanAccessStudent(current, ownerOf(existing), s.perms, "student:update:any") {
		return helper.NewError(fiber.StatusForbidden, "forbidden", "tidak berhak mengubah data student ini")
	}

	var req model.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
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
		return translateError(err)
	}

	return helper.Success(c, fiber.StatusOK, "student updated successfully", updated)
}

// Patch handles PATCH /api/v1/students/:id
func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "belum terautentikasi")
	}

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID format")
	}

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err)
	}

	// Ownership check
	if !CanAccessStudent(current, ownerOf(existing), s.perms, "student:update:any") {
		return helper.NewError(fiber.StatusForbidden, "forbidden", "tidak berhak mengubah data student ini")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}

	if IsEmptyPatch(req) {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "no fields to update")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	updated := ApplyPatch(existing, req)

	result, err := s.repo.Update(ctx, updated)
	if err != nil {
		return translateError(err)
	}

	return helper.Success(c, fiber.StatusOK, "student patched successfully", result)
}

// Delete handles DELETE /api/v1/students/:id
func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID format")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(err)
	}

	return helper.NoContent(c)
}

// ============================================================================
// ERROR TRANSLATION
// ============================================================================

func translateError(err error) *helper.AppError {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NewError(fiber.StatusNotFound, "not_found", "data not found")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.NewError(fiber.StatusConflict, "conflict", "NIM already exists")
	default:
		return helper.InternalError(err)
	}
}
