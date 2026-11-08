package service_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v2"

	"api-students/app/service"
	"api-students/helper"
)

// TestTranslateReviewError menguji pemetaan seluruh sentinel error review
// menuju *helper.AppError (status + kode stabil), termasuk error yang
// dibungkus (wrapped) dan error tak dikenal.
func TestTranslateReviewError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"invalid id", service.ErrInvalidID, fiber.StatusBadRequest, "bad_request"},
		{"not found", service.ErrNotFound, fiber.StatusNotFound, "not_found"},
		{"duplicate", service.ErrDuplicate, fiber.StatusConflict, "duplicate"},
		{"not owner", service.ErrNotOwner, fiber.StatusForbidden, "forbidden"},
		{"own review", service.ErrOwnReview, fiber.StatusForbidden, "forbidden"},
		{"already voted", service.ErrAlreadyVoted, fiber.StatusConflict, "already_voted"},
		{"reply limit", service.ErrReplyLimit, fiber.StatusConflict, "reply_limit"},
		{
			name:       "wrapped error",
			err:        fmt.Errorf("query review: %w", service.ErrNotFound),
			wantStatus: fiber.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:       "unknown error",
			err:        errors.New("koneksi mongo putus"),
			wantStatus: fiber.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := service.TranslateReviewError(tc.err)

			var appErr *helper.AppError
			if !errors.As(got, &appErr) {
				t.Fatalf("hasil harus *helper.AppError, dapat %T (%v)", got, got)
			}
			if appErr.Status != tc.wantStatus {
				t.Errorf("status = %d, want %d", appErr.Status, tc.wantStatus)
			}
			if appErr.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", appErr.Code, tc.wantCode)
			}
		})
	}

	t.Run("nil", func(t *testing.T) {
		if err := service.TranslateReviewError(nil); err != nil {
			t.Errorf("TranslateReviewError(nil) = %v, want nil", err)
		}
	})
}
