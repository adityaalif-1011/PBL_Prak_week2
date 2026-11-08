package helper

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// AppError adalah tipe error tunggal untuk seluruh API.
// Status  = HTTP status code
// Code    = kode stabil yang boleh dikonsumsi client
// Message = pesan aman untuk ditampilkan ke client
// Fields  = detail per-field untuk error validasi (422)
// cause   = error asli, hanya untuk keperluan log (private)
type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
	cause   error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s (%d): %s: %v", e.Code, e.Status, e.Message, e.cause)
	}
	return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Message)
}

func (e *AppError) Unwrap() error { return e.cause }

// NewError membuat AppError tanpa cause.
func NewError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// WrapError membuat AppError yang membungkus error asli (untuk log 5xx).
func WrapError(status int, code, message string, cause error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, cause: cause}
}

// InternalError membungkus error tak terduga menjadi 500.
func InternalError(cause error) *AppError {
	return WrapError(fiber.StatusInternalServerError, "internal_error", "internal server error", cause)
}

// ValidationFailed membuat 422 dengan detail per-field.
func ValidationFailed(fields map[string]string) *AppError {
	return &AppError{
		Status:  fiber.StatusUnprocessableEntity,
		Code:    "validation_failed",
		Message: "validation failed",
		Fields:  fields,
	}
}

// StatusOf mengambil HTTP status dari error; selain AppError dianggap 500.
func StatusOf(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return fiber.StatusInternalServerError
}

// ErrorResponse adalah body JSON yang dikirim untuk semua error.
type ErrorResponse struct {
	Success   bool              `json:"success"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}
