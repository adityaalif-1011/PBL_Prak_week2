package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"api-students/app/repository"
	"api-students/helper"
)

// Sentinel error domain review. Sebagian adalah alias error repository agar
// satu kegagalan selalu punya satu arti di seluruh lapisan (repository →
// service → HTTP), dan bisa diuji lewat errors.Is.
var (
	// ErrInvalidID - path param bukan ID yang valid.
	ErrInvalidID = errors.New("invalid review id")
	// ErrNotFound - review tidak ada (alias repository.ErrReviewNotFound).
	ErrNotFound = repository.ErrReviewNotFound
	// ErrDuplicate - student sudah mengulas kelas yang sama (unique index).
	ErrDuplicate = repository.ErrDuplicate
	// ErrNotOwner - review bukan milik pemanggil (untuk akses ubah/hapus review).
	ErrNotOwner = errors.New("review does not belong to this student")
	// ErrOwnReview - menandai review milik sendiri sebagai helpful (self-vote).
	ErrOwnReview = errors.New("cannot mark your own review as helpful")
	// ErrAlreadyVoted - student sudah menandai review helpful (alias repository.ErrAlreadyHelpful).
	ErrAlreadyVoted = repository.ErrAlreadyHelpful
	// ErrReplyLimit - kuota balasan pada satu review sudah penuh.
	ErrReplyLimit = repository.ErrReplyLimit
)

// TranslateReviewError menerjemahkan error domain review menjadi AppError
// (status + kode stabil) yang siap dikirim ke client.
//
// Bila error tidak dikenal, hasilnya AppError 500 yang tetap membungkus error
// aslinya (untuk log), sehingga pemanggil cukup menulis `return
// TranslateReviewError(err)` di setiap cabang error.
func TranslateReviewError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrInvalidID):
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "invalid id")

	case errors.Is(err, ErrNotFound):
		return helper.NewError(fiber.StatusNotFound, "not_found", "review not found")

	case errors.Is(err, ErrDuplicate):
		return helper.NewError(fiber.StatusConflict, "duplicate",
			"student already reviewed this course")

	case errors.Is(err, ErrNotOwner):
		return helper.NewError(fiber.StatusForbidden, "forbidden",
			"review does not belong to this student")

	case errors.Is(err, ErrOwnReview):
		return helper.NewError(fiber.StatusForbidden, "forbidden",
			"cannot mark your own review as helpful")

	case errors.Is(err, ErrAlreadyVoted):
		return helper.NewError(fiber.StatusConflict, "already_voted",
			"review already marked helpful")

	case errors.Is(err, ErrReplyLimit):
		return helper.NewError(fiber.StatusConflict, "reply_limit", "reply limit reached")

	case errors.Is(err, repository.ErrCourseNotFound):
		return helper.NewError(fiber.StatusNotFound, "not_found", "course not found")

	case errors.Is(err, repository.ErrEnrollmentNotFound):
		return helper.NewError(fiber.StatusForbidden, "not_enrolled",
			"student is not enrolled in this course")

	default:
		return helper.InternalError(err)
	}
}
