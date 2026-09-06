package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/service"
	"api-students/helper"
	"api-students/middleware"
	"api-students/route"
)

// NewApp merakit aplikasi
func NewApp(pool *pgxpool.Pool, studentService *service.StudentService) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: GetEnv("APP_NAME", "API Students"),
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return helper.Fail(c, fiber.StatusInternalServerError, "internal server error")
		},
	})

	// Middleware global
	middleware.Register(app)

	// Route
	route.Register(app, pool, studentService)

	// 404 handler
	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint not found")
	})

	return app
}
