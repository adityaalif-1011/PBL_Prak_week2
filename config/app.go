package config

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/helper"
	"api-students/middleware"
	"api-students/route"
)

func NewApp(pool *pgxpool.Pool, deps route.Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "API Students"),
		BodyLimit:    1 * 1024 * 1024, // 1 MB
		ErrorHandler: newErrorHandler(),
	})

	middleware.Register(app)
	route.Register(app, deps)

	app.Use(func(c *fiber.Ctx) error {
		return helper.NewError(fiber.StatusNotFound, "not_found", "endpoint not found")
	})

	return app
}

// newErrorHandler mengubah setiap error menjadi JSON ErrorResponse berisi request_id.
// 4xx dicatat sebagai WARN, 5xx sebagai ERROR.
func newErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		requestID, _ := c.Locals("requestid").(string)

		appErr := toAppError(err)

		attrs := []any{
			slog.Int("status", appErr.Status),
			slog.String("code", appErr.Code),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.String("request_id", requestID),
		}
		if len(appErr.Fields) > 0 {
			attrs = append(attrs, slog.Any("fields", appErr.Fields))
		}

		if appErr.Status >= fiber.StatusInternalServerError {
			attrs = append(attrs, slog.String("error", err.Error()))
			slog.Error("request gagal", attrs...)
		} else {
			slog.Warn("request ditolak", attrs...)
		}

		return c.Status(appErr.Status).JSON(helper.ErrorResponse{
			Success:   false,
			Code:      appErr.Code,
			Message:   appErr.Message,
			RequestID: requestID,
			Fields:    appErr.Fields,
		})
	}
}

// toAppError menurunkan error apapun menjadi AppError.
func toAppError(err error) *helper.AppError {
	var appErr *helper.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return helper.NewError(fiberErr.Code, fiberCode(fiberErr.Code), fiberErr.Message)
	}

	return helper.WrapError(fiber.StatusInternalServerError, "internal_error", "internal server error", err)
}

func fiberCode(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "bad_request"
	case fiber.StatusUnauthorized:
		return "unauthorized"
	case fiber.StatusForbidden:
		return "forbidden"
	case fiber.StatusNotFound:
		return "not_found"
	case fiber.StatusMethodNotAllowed:
		return "method_not_allowed"
	case fiber.StatusConflict:
		return "conflict"
	case fiber.StatusTooManyRequests:
		return "rate_limited"
	case fiber.StatusUnprocessableEntity:
		return "validation_failed"
	case fiber.StatusUnsupportedMediaType:
		return "unsupported_media_type"
	default:
		return "error"
	}
}
