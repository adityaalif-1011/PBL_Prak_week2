package service

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

// ReviewService menangani ulasan course (MongoDB).
type ReviewService struct {
	reviews repository.ReviewRepository
	gate    repository.ReviewGate
}

// NewReviewService membuat service review.
// reviews: penyimpanan review (MongoDB, bisa fake).
// gate: cek prasyarat review (kelas ada & student peserta).
func NewReviewService(reviews repository.ReviewRepository, gate repository.ReviewGate) *ReviewService {
	return &ReviewService{reviews: reviews, gate: gate}
}

// Create - POST /api/v1/courses/:courseID/reviews.
// Prasyarat: kelas harus ada dan student harus pesertanya; rating1..5.
// @Summary Tulis review kelas
// @Description Menambah ulasan mahasiswa untuk sebuah kelas. Kelas harus ada dan student harus terdaftar sebagai peserta; rating 1..5; tag dinormalkan (huruf kecil, tanpa duplikat).
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path int true "ID course"
// @Param body body model.CreateReviewRequest true "Isi review"
// @Success 201 {object} model.ResponseEnvelope{data=model.Review}
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 403 {object} helper.ErrorResponse
// @Failure 404 {object} helper.ErrorResponse
// @Failure 409 {object} helper.ErrorResponse
// @Failure 422 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /courses/{id}/reviews [post]
func (s *ReviewService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, ok := helper.ParamID(c)
	if !ok {
		return TranslateReviewError(ErrInvalidID)
	}

	var req model.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	req.CourseID = courseID
	req.Tags = NormalizeTags(req.Tags)
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	if err := s.gate.CourseExists(ctx, courseID); err != nil {
		return TranslateReviewError(err)
	}
	if err := s.gate.IsEnrolled(ctx, req.StudentID, courseID); err != nil {
		return TranslateReviewError(err)
	}

	created, err := s.reviews.Create(ctx, req)
	if err != nil {
		return TranslateReviewError(err)
	}
	return helper.Created(c, "review created successfully", created,
		"/api/v1/courses/"+strconv.Itoa(courseID)+"/reviews")
}

// ListByCourse - GET /api/v1/courses/:courseID/reviews.
// Mengembalikan model.ReviewPage (daftar + metadata halaman).
// @Summary Daftar review satu kelas
// @Description Mengambil seluruh ulasan sebuah kelas (terbaru di atas) beserta metadata halamannya.
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path int true "ID course"
// @Success 200 {object} model.ResponseEnvelope{data=model.ReviewPage}
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 500 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /courses/{id}/reviews [get]
func (s *ReviewService) ListByCourse(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, ok := helper.ParamID(c)
	if !ok {
		return TranslateReviewError(ErrInvalidID)
	}

	list, err := s.reviews.FindByCourse(ctx, courseID)
	if err != nil {
		return TranslateReviewError(err)
	}
	if list == nil {
		list = []model.Review{}
	}

	return helper.Success(c, fiber.StatusOK, "reviews retrieved successfully",
		model.ReviewPage{
			CourseID: courseID,
			Reviews:  list,
			Meta:     model.ReviewPageMeta{Total: len(list)},
		})
}

// Summary - GET /api/v1/courses/:courseID/reviews/summary.
// @Summary Ringkasan review satu kelas
// @Description Mengambil agregasi review sebuah kelas: total, rata-rata rating, distribusi rating, jumlah balasan, dan jumlah helpful.
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path int true "ID course"
// @Success 200 {object} model.ResponseEnvelope{data=model.ReviewSummary}
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 500 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /courses/{id}/reviews/summary [get]
func (s *ReviewService) Summary(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, ok := helper.ParamID(c)
	if !ok {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid course ID")
	}

	summary, err := s.reviews.Summary(ctx, courseID)
	if err != nil {
		return helper.InternalError(err)
	}
	return helper.Success(c, fiber.StatusOK, "review summary retrieved", summary)
}

// AddReply - POST /api/v1/reviews/:id/replies.
// @Summary Balas sebuah review
// @Description Menambahkan balasan pada review tertentu. Dibatasi jumlah balasan per review (reply_limit).
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path string true "ID review (ObjectId MongoDB)"
// @Param body body model.AddReplyRequest true "Isi balasan"
// @Success 200 {object} model.ResponseEnvelope{data=model.Review}
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 404 {object} helper.ErrorResponse
// @Failure 409 {object} helper.ErrorResponse
// @Failure 422 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /reviews/{id}/replies [post]
func (s *ReviewService) AddReply(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	reviewID, ok := helper.ObjectIDParam(c, "id")
	if !ok {
		return TranslateReviewError(ErrInvalidID)
	}

	var req model.AddReplyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	updated, err := s.reviews.AddReply(ctx, reviewID, req)
	if err != nil {
		return TranslateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "reply added", updated)
}

// MarkHelpful - POST /api/v1/reviews/:id/helpful.
// @Summary Tandai review membantu
// @Description Menambahkan (atau menghapus) student pada daftar helpful sebuah review — satu student hanya boleh menandai satu kali.
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path string true "ID review (ObjectId MongoDB)"
// @Param body body model.MarkHelpfulRequest true "student_id penanda"
// @Success 200 {object} model.ResponseEnvelope{data=model.Review}
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 404 {object} helper.ErrorResponse
// @Failure 422 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /reviews/{id}/helpful [post]
func (s *ReviewService) MarkHelpful(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	reviewID, ok := helper.ObjectIDParam(c, "id")
	if !ok {
		return TranslateReviewError(ErrInvalidID)
	}

	var req model.MarkHelpfulRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid JSON body")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	updated, err := s.reviews.MarkHelpful(ctx, reviewID, req.StudentID)
	if err != nil {
		return TranslateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "review marked helpful", updated)
}
