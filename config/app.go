package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/helper"
	"api-students/middleware"
	"api-students/route"
)

func NewApp(pool *pgxpool.Pool, deps route.Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:   GetEnv("APP_NAME", "API Students"),
		BodyLimit: 1 * 1024 * 1024, // 1 MB
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return helper.Fail(c, fiber.StatusInternalServerError, "internal server error")
		},
	})

	middleware.Register(app)
	route.Register(app, deps)

	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint not found")
	})

	return app
}
