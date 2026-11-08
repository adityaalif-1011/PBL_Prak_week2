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

// CourseService menangani course & enrollment (Modul 8 & 9 & 11).
type CourseService struct {
	courses *repository.CourseRepository
	tx      *repository.TxManager
	outbox  repository.OutboxRepository
}

func NewCourseService(courses *repository.CourseRepository, tx *repository.TxManager, outbox repository.OutboxRepository) *CourseService {
	return &CourseService{courses: courses, tx: tx, outbox: outbox}
}

// enrollmentEventPayload - payload event outbox untuk notifikasi enrollment.
type enrollmentEventPayload struct {
	StudentID     int    `json:"student_id"`
	CourseID      int    `json:"course_id"`
	CourseKode    string `json:"course_kode"`
	CourseNama    string `json:"course_nama"`
	TahunAkademik string `json:"tahun_akademik"`
	OldCourseID   int    `json:"old_course_id,omitempty"`
	OldCourseKode string `json:"old_course_kode,omitempty"`
	OldCourseNama string `json:"old_course_nama,omitempty"`
}

// ============================================================
// COURSES
// ============================================================

// List - GET /api/v1/courses (menampilkan sisa_kuota).
// @Summary Daftar mata kuliah
// @Description Mengambil seluruh course beserta sisa kuotanya.
// @Tags courses
// @Accept json
// @Produce json
// @Success 200 {object} model.ResponseEnvelope{data=[]model.Course}
// @Failure 401 {object} helper.ErrorResponse
// @Failure 500 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /courses [get]
func (s *CourseService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courses, err := s.courses.FindAll(ctx)
	if err != nil {
		return helper.InternalError(err)
	}
	return helper.SuccessList(c, "courses retrieved successfully", courses, nil)
}

// Get - GET /api/v1/courses/:id.
// @Summary Detail mata kuliah
// @Description Mengambil satu course berdasarkan id.
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "ID course"
// @Success 200 {object} model.ResponseEnvelope{data=model.Course}
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 404 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /courses/{id} [get]
func (s *CourseService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID format")
	}

	course, err := s.courses.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCourseNotFound) {
			return helper.NewError(fiber.StatusNotFound, "not_found", "course not found")
		}
		return helper.InternalError(err)
	}
	return helper.Success(c, fiber.StatusOK, "course retrieved successfully", course)
}

// Mine - GET /api/v1/courses/mine (auth).
// Mengembalikan model.MyCourses: daftar mata kuliah yang diikuti student milik
// user yang login, plus total & batas SKS-nya.
// @Summary Mata kuliah milik saya
// @Description Mengambil course yang diikuti student milik user yang login, lengkap dengan total & batas SKS.
// @Tags courses
// @Accept json
// @Produce json
// @Success 200 {object} model.ResponseEnvelope{data=model.MyCourses}
// @Failure 401 {object} helper.ErrorResponse
// @Failure 500 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /courses/mine [get]
func (s *CourseService) Mine(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "belum terautentikasi")
	}

	mine, err := s.courses.FindMyCourses(ctx, current.UserID)
	if err != nil {
		return helper.InternalError(err)
	}
	return helper.Success(c, fiber.StatusOK, "my courses retrieved successfully", mine)
}

// Create - POST /api/v1/courses.
func (s *CourseService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateCourseRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	created, err := s.courses.Create(ctx, req)
	if err != nil {
		return helper.InternalError(err)
	}
	return helper.Created(c, "course created successfully", created,
		"/api/v1/courses/"+strconv.Itoa(created.ID))
}

// UpdateQuota - PUT /api/v1/courses/:id/quota (optimistic lock via version).
// @Summary Ubah kuota course
// @Description Mengubah kuota dengan optimistic locking: kolom version harus cocok, lalu version dinaikkan.
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "ID course"
// @Param body body model.UpdateQuotaRequest true "Kuota baru + versi terakhir"
// @Success 200 {object} model.ResponseEnvelope{data=model.Course}
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 404 {object} helper.ErrorResponse
// @Failure 409 {object} helper.ErrorResponse
// @Failure 422 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /courses/{id}/quota [put]
func (s *CourseService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID format")
	}

	var req model.UpdateQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	updated, err := s.courses.UpdateQuota(ctx, id, req.Kuota, req.Version)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrVersionConflict):
			return helper.NewError(fiber.StatusConflict, "version_conflict",
				"kuota sudah diubah orang lain, muat ulang lalu coba lagi")
		case errors.Is(err, repository.ErrCourseNotFound):
			return helper.NewError(fiber.StatusNotFound, "not_found", "course not found")
		default:
			return helper.InternalError(err)
		}
	}
	return helper.Success(c, fiber.StatusOK, "quota updated successfully", updated)
}

// ============================================================
// ENROLL / DROP / SWAP (satu transaksi)
// ============================================================

// Enroll - POST /api/v1/enrollments.
// Cek duplikasi, kuota, dan batas SKS SEMUA di dalam satu transaksi.
// @Summary Daftar mata kuliah
// @Description Mendaftarkan student ke sebuah course: cek duplikasi, kuota, dan batas SKS dalam SATU transaksi, sekaligus mencatat event ke outbox.
// @Tags enrollments
// @Accept json
// @Produce json
// @Param body body model.CreateEnrollmentRequest true "Data pendaftaran"
// @Success 201 {object} model.ResponseEnvelope{data=model.Enrollment}
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 404 {object} helper.ErrorResponse
// @Failure 409 {object} helper.ErrorResponse
// @Failure 422 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /enrollments [post]
func (s *CourseService) Enroll(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateEnrollmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	var created model.Enrollment
	err := s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		repo := s.courses.WithTx(tx)

		// 1) kunci student (FOR UPDATE) untuk cek batas SKS
		student, err := repo.LockStudent(ctx, req.StudentID)
		if err != nil {
			if errors.Is(err, repository.ErrStudentNotFound) {
				return helper.NewError(fiber.StatusNotFound, "not_found", "student not found")
			}
			return helper.InternalError(err)
		}

		// 2) kunci course (FOR UPDATE) untuk cek kuota
		course, err := repo.LockCourse(ctx, req.CourseID)
		if err != nil {
			if errors.Is(err, repository.ErrCourseNotFound) {
				return helper.NewError(fiber.StatusNotFound, "not_found", "course not found")
			}
			return helper.InternalError(err)
		}

		// 3) cek duplikasi
		if _, err := repo.FindEnrollment(ctx, req.StudentID, req.CourseID, req.TahunAkademik); err == nil {
			return helper.NewError(fiber.StatusConflict, "conflict", "student already enrolled in this course")
		} else if !errors.Is(err, repository.ErrEnrollmentNotFound) {
			return helper.InternalError(err)
		}

		// 4) cek kuota
		if course.EnrolledCount >= course.Kuota {
			return helper.NewError(fiber.StatusConflict, "course_full", "course quota is full")
		}

		// 5) cek batas SKS (aturan murni: WithinCreditLimit)
		if !WithinCreditLimit(student.TotalSKS, course.SKS, student.BatasSKS) {
			return helper.NewError(fiber.StatusConflict, "sks_limit_exceeded",
				"adding this course would exceed the SKS limit")
		}

		// 6) daftar + tambah enrolled_count + tambah total_sks
		created, err = repo.CreateEnrollment(ctx, req.StudentID, req.CourseID, req.TahunAkademik)
		if err != nil {
			if errors.Is(err, repository.ErrEnrollmentDuplicate) {
				return helper.NewError(fiber.StatusConflict, "conflict", "student already enrolled")
			}
			return helper.InternalError(err)
		}
		if err := repo.IncrementEnrolled(ctx, req.CourseID); err != nil {
			return helper.InternalError(err)
		}
		if err := repo.UpdateStudentSKS(ctx, req.StudentID, course.SKS); err != nil {
			return helper.InternalError(err)
		}

		// 7) CATAT EVENT outbox di dalam transaksi yang sama (transactional outbox).
		// Bila transaksi ini rollback, event ikut hilang; bila commit, event pasti
		// ada dan akan direlay oleh OutboxRelay.
		if _, err := s.outbox.Insert(ctx, tx, model.EventEnrollmentCreated, enrollmentEventPayload{
			StudentID:     req.StudentID,
			CourseID:      req.CourseID,
			CourseKode:    course.Kode,
			CourseNama:    course.Nama,
			TahunAkademik: req.TahunAkademik,
		}); err != nil {
			return helper.InternalError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return helper.Created(c, "enrollment created successfully", created,
		"/api/v1/enrollments/"+strconv.Itoa(created.ID))
}

// Drop - DELETE /api/v1/enrollments/:id (mengembalikan kuota).
// @Summary Keluar dari mata kuliah
// @Description Menghapus enrollment, mengembalikan kuota & SKS, lalu mencatat event outbox dalam transaksi yang sama.
// @Tags enrollments
// @Accept json
// @Produce json
// @Param id path int true "ID enrollment"
// @Success 204 {object} helper.ErrorResponse
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 404 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /enrollments/{id} [delete]
func (s *CourseService) Drop(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid ID format")
	}

	err := s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		repo := s.courses.WithTx(tx)

		// 1) ambil info enrollment (course & sks)
		enrollment, err := repo.FindEnrollmentByID(ctx, id)
		if err != nil {
			if errors.Is(err, repository.ErrEnrollmentNotFound) {
				return helper.NewError(fiber.StatusNotFound, "not_found", "enrollment not found")
			}
			return helper.InternalError(err)
		}

		// 2) kunci course lalu hapus enrollment + kurangi enrolled + kurangi SKS
		if _, err := repo.LockCourse(ctx, enrollment.CourseID); err != nil {
			return helper.InternalError(err)
		}
		if err := repo.DeleteEnrollment(ctx, id); err != nil {
			return helper.InternalError(err)
		}
		if err := repo.DecrementEnrolled(ctx, enrollment.CourseID); err != nil {
			return helper.InternalError(err)
		}
		if err := repo.UpdateStudentSKS(ctx, enrollment.StudentID, -enrollment.SKS); err != nil {
			return helper.InternalError(err)
		}

		// CATAT EVENT outbox di dalam transaksi yang sama.
		if _, err := s.outbox.Insert(ctx, tx, model.EventEnrollmentDropped, enrollmentEventPayload{
			StudentID:     enrollment.StudentID,
			CourseID:      enrollment.CourseID,
			CourseKode:    enrollment.Kode,
			CourseNama:    enrollment.Nama,
			TahunAkademik: enrollment.TahunAkademik,
		}); err != nil {
			return helper.InternalError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return helper.NoContent(c)
}

// Swap - POST /api/v1/enrollments/swap (drop lama + enroll baru, satu transaksi).
// @Summary Pindah kelas
// @Description Menukar enrollment lama dengan kelas baru dalam SATU transaksi (drop + enroll), termasuk penyesuaian kuota, SKS, dan event outbox.
// @Tags enrollments
// @Accept json
// @Produce json
// @Param body body model.SwapEnrollmentRequest true "Kelas lama dan kelas baru"
// @Success 201 {object} model.ResponseEnvelope{data=model.Enrollment}
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 404 {object} helper.ErrorResponse
// @Failure 409 {object} helper.ErrorResponse
// @Failure 422 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /enrollments/swap [post]
func (s *CourseService) Swap(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.SwapEnrollmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	var created model.Enrollment
	err := s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		repo := s.courses.WithTx(tx)

		student, err := repo.LockStudent(ctx, req.StudentID)
		if err != nil {
			if errors.Is(err, repository.ErrStudentNotFound) {
				return helper.NewError(fiber.StatusNotFound, "not_found", "student not found")
			}
			return helper.InternalError(err)
		}

		// Kunci kedua course dalam urutan id tetap untuk hindari deadlock.
		courses, err := repo.LockCourses(ctx, []int{req.OldCourseID, req.NewCourseID})
		if err != nil {
			return helper.InternalError(err)
		}
		oldCourse, okOld := courses[req.OldCourseID]
		newCourse, okNew := courses[req.NewCourseID]
		if !okOld {
			return helper.NewError(fiber.StatusNotFound, "not_found", "old course not found")
		}
		if !okNew {
			return helper.NewError(fiber.StatusNotFound, "not_found", "new course not found")
		}
		if req.OldCourseID == req.NewCourseID {
			return helper.NewError(fiber.StatusBadRequest, "bad_request", "old and new course must differ")
		}

		// Pastikan enrollment lama ada.
		if _, err := repo.FindEnrollment(ctx, req.StudentID, req.OldCourseID, req.TahunAkademik); err != nil {
			if errors.Is(err, repository.ErrEnrollmentNotFound) {
				return helper.NewError(fiber.StatusNotFound, "not_found", "old enrollment not found")
			}
			return helper.InternalError(err)
		}
		// Pastikan belum terdaftar di course baru.
		if _, err := repo.FindEnrollment(ctx, req.StudentID, req.NewCourseID, req.TahunAkademik); err == nil {
			return helper.NewError(fiber.StatusConflict, "conflict", "already enrolled in new course")
		} else if !errors.Is(err, repository.ErrEnrollmentNotFound) {
			return helper.InternalError(err)
		}

		// Kuota course baru harus tersedia.
		if newCourse.EnrolledCount >= newCourse.Kuota {
			return helper.NewError(fiber.StatusConflict, "course_full", "new course quota is full")
		}
		// Total SKS setelah pindah kelas tidak boleh melebihi batas
		// (aturan murni: CreditsAfterSwap).
		totalAfter := CreditsAfterSwap(student.TotalSKS, oldCourse.SKS, newCourse.SKS)
		if totalAfter > student.BatasSKS {
			return helper.NewError(fiber.StatusConflict, "sks_limit_exceeded",
				"swapping would exceed the SKS limit")
		}
		delta := newCourse.SKS - oldCourse.SKS

		// Drop lama.
		if err := repo.DeleteEnrollmentByKeys(ctx, req.StudentID, req.OldCourseID, req.TahunAkademik); err != nil {
			return helper.InternalError(err)
		}
		if err := repo.DecrementEnrolled(ctx, req.OldCourseID); err != nil {
			return helper.InternalError(err)
		}
		// Enroll baru.
		created, err = repo.CreateEnrollment(ctx, req.StudentID, req.NewCourseID, req.TahunAkademik)
		if err != nil {
			if errors.Is(err, repository.ErrEnrollmentDuplicate) {
				return helper.NewError(fiber.StatusConflict, "conflict", "student already enrolled")
			}
			return helper.InternalError(err)
		}
		if err := repo.IncrementEnrolled(ctx, req.NewCourseID); err != nil {
			return helper.InternalError(err)
		}
		// Sesuaikan total SKS dengan delta.
		if delta != 0 {
			if err := repo.UpdateStudentSKS(ctx, req.StudentID, delta); err != nil {
				return helper.InternalError(err)
			}
		}

		// CATAT EVENT outbox di dalam transaksi yang sama.
		if _, err := s.outbox.Insert(ctx, tx, model.EventEnrollmentSwapped, enrollmentEventPayload{
			StudentID:     req.StudentID,
			CourseID:      req.NewCourseID,
			CourseKode:    newCourse.Kode,
			CourseNama:    newCourse.Nama,
			TahunAkademik: req.TahunAkademik,
			OldCourseID:   req.OldCourseID,
			OldCourseKode: oldCourse.Kode,
			OldCourseNama: oldCourse.Nama,
		}); err != nil {
			return helper.InternalError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return helper.Created(c, "enrollment swapped successfully", created, "")
}
