package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/service"
	"api-students/helper"
	"api-students/middleware"
	"api-students/route"
)

func NewApp(
	pool *pgxpool.Pool,
	studentService *service.StudentService,
	achievementService *service.AchievementService, // ← TAMBAH INI
) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: GetEnv("APP_NAME", "API Students"),
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return helper.Fail(c, fiber.StatusInternalServerError, "internal server error")
		},
	})

	middleware.Register(app)
	route.Register(app, pool, studentService, achievementService) // ← UBAH JADI 4 ARGUMEN

	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint not found")
	})

	return app
}
