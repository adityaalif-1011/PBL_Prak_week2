package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

// SupervisorService menangani pembimbing TA (Modul 9).
type SupervisorService struct {
	supervisors *repository.SupervisorRepository
	students    repository.StudentRepository
	tx          *repository.TxManager
}

func NewSupervisorService(
	supervisors *repository.SupervisorRepository,
	students repository.StudentRepository,
	tx *repository.TxManager,
) *SupervisorService {
	return &SupervisorService{supervisors: supervisors, students: students, tx: tx}
}

// Create - POST /api/v1/supervisors.
func (s *SupervisorService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	created, err := s.supervisors.Create(ctx, req)
	if err != nil {
		return helper.InternalError(err)
	}
	return helper.Created(c, "supervisor created successfully", created,
		"/api/v1/supervisors/"+strconv.Itoa(created.ID))
}

// Get - GET /api/v1/supervisors/:id.
func (s *SupervisorService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID format")
	}

	sup, err := s.supervisors.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrSupervisorNotFound) {
			return helper.NewError(fiber.StatusNotFound, "not_found", "supervisor not found")
		}
		return helper.InternalError(err)
	}
	return helper.Success(c, fiber.StatusOK, "supervisor found", sup)
}

// Assign - POST /api/v1/supervisors/:id/assign (satu transaksi).
// Mengunci supervisor (FOR UPDATE) untuk cek kapasitas secara aman.
func (s *SupervisorService) Assign(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	supervisorID, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid supervisor ID")
	}

	var req model.AssignSupervisionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	// Pastikan student ada.
	if _, err := s.students.FindByID(ctx, req.StudentID); err != nil {
		return helper.NewError(fiber.StatusNotFound, "not_found", "student not found")
	}

	var created model.Supervision
	err := s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		repo := s.supervisors.WithTx(tx)

		// Kunci supervisor (FOR UPDATE) untuk cek kapasitas.
		sup, err := repo.LockSupervisor(ctx, supervisorID)
		if err != nil {
			if errors.Is(err, repository.ErrSupervisorNotFound) {
				return helper.NewError(fiber.StatusNotFound, "not_found", "supervisor not found")
			}
			return helper.InternalError(err)
		}

		count, err := repo.CountAssigned(ctx, supervisorID)
		if err != nil {
			return helper.InternalError(err)
		}
		if count >= sup.MaxStudents {
			return helper.NewError(fiber.StatusConflict, "supervisor_full",
				"supervisor has reached max students")
		}

		created, err = repo.Assign(ctx, supervisorID, req.StudentID)
		if err != nil {
			if errors.Is(err, repository.ErrAlreadySupervised) {
				return helper.NewError(fiber.StatusConflict, "conflict",
					"student already has a supervisor")
			}
			return helper.InternalError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return helper.Created(c, "student assigned to supervisor", created, "")
}

// Switch - POST /api/v1/supervisions/switch (satu transaksi).
func (s *SupervisorService) Switch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.SwitchSupervisionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	var updated model.Supervision
	err := s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		repo := s.supervisors.WithTx(tx)

		// Pastikan student sedang punya pembimbing.
		if _, err := repo.FindByStudent(ctx, req.StudentID); err != nil {
			if errors.Is(err, repository.ErrSupervisionNotFound) {
				return helper.NewError(fiber.StatusNotFound, "not_found",
					"student is not assigned to any supervisor")
			}
			return helper.InternalError(err)
		}

		// Kunci pembimbing baru (FOR UPDATE) untuk cek kapasitas.
		sup, err := repo.LockSupervisor(ctx, req.SupervisorID)
		if err != nil {
			if errors.Is(err, repository.ErrSupervisorNotFound) {
				return helper.NewError(fiber.StatusNotFound, "not_found", "new supervisor not found")
			}
			return helper.InternalError(err)
		}
		count, err := repo.CountAssigned(ctx, req.SupervisorID)
		if err != nil {
			return helper.InternalError(err)
		}
		if count >= sup.MaxStudents {
			return helper.NewError(fiber.StatusConflict, "supervisor_full",
				"new supervisor has reached max students")
		}

		updated, err = repo.Switch(ctx, req.StudentID, req.SupervisorID)
		if err != nil {
			return helper.InternalError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return helper.Success(c, fiber.StatusOK, "supervision switched successfully", updated)
}

// Patch - PATCH /api/v1/supervisors/:id.
func (s *SupervisorService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID format")
	}

	var req model.PatchSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if req.Name == nil && req.MaxStudents == nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "no fields to update")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	updated, err := s.supervisors.Patch(ctx, id, req)
	if err != nil {
		if errors.Is(err, repository.ErrSupervisorNotFound) {
			return helper.NewError(fiber.StatusNotFound, "not_found", "supervisor not found")
		}
		return helper.InternalError(err)
	}
	return helper.Success(c, fiber.StatusOK, "supervisor updated successfully", updated)
}
