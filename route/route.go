package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/service"
	"api-students/helper"
	"api-students/middleware"
)

func Register(
	app *fiber.App,
	pool *pgxpool.Pool,
	studentService *service.StudentService,
	achievementService *service.AchievementService, // ← TAMBAH INI
) {
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", healthCheck(pool))

	// ============================================================
	// STUDENT ROUTES
	// ============================================================
	students := api.Group("/students", middleware.RequireJSON)
	students.Get("/", studentService.List)
	students.Get("/:id", studentService.Get)
	students.Post("/", studentService.Create)
	students.Put("/:id", studentService.Replace)
	students.Patch("/:id", studentService.Patch)
	students.Delete("/:id", studentService.Delete)

	// ============================================================
	// ACHIEVEMENT ROUTES
	// ============================================================

	// Nested: prestasi milik student
	students.Get("/:id/achievements", achievementService.GetByStudentID)

	// Standalone
	achievements := api.Group("/achievements", middleware.RequireJSON)
	achievements.Get("/:id", achievementService.Get)
	achievements.Post("/", achievementService.Create)
	achievements.Put("/:id", achievementService.Update)
	achievements.Patch("/:id", achievementService.Patch)
	achievements.Delete("/:id", achievementService.Delete)
}

func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable, "database is unavailable")
		}
		return helper.Success(c, fiber.StatusOK, "server and database are running", nil)
	}
}
