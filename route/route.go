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

// Dependencies - kumpulan dependency
type Dependencies struct {
	Pool               *pgxpool.Pool
	JWT                *helper.JWTManager
	StudentService     *service.StudentService
	AchievementService *service.AchievementService
	AuthService        *service.AuthService
}

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	// ============================================================
	// HEALTH (PUBLIK)
	// ============================================================
	api.Get("/health", healthCheck(deps.Pool))

	// ============================================================
	// AUTH (PUBLIK)
	// ============================================================
	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	// ============================================================
	// STUDENTS (WAJIB LOGIN)
	// ============================================================
	students := api.Group("/students",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT),
	)
	students.Get("/", deps.StudentService.List)
	students.Get("/:id", deps.StudentService.Get)
	students.Post("/", deps.StudentService.Create)
	students.Put("/:id", deps.StudentService.Replace)
	students.Patch("/:id", deps.StudentService.Patch)
	students.Delete("/:id", deps.StudentService.Delete)

	// ============================================================
	// ACHIEVEMENTS (WAJIB LOGIN)
	// ============================================================
	students.Get("/:id/achievements", deps.AchievementService.GetByStudentID)

	achievements := api.Group("/achievements",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT),
	)
	achievements.Get("/:id", deps.AchievementService.Get)
	achievements.Post("/", deps.AchievementService.Create)
	achievements.Put("/:id", deps.AchievementService.Update)
	achievements.Patch("/:id", deps.AchievementService.Patch)
	achievements.Delete("/:id", deps.AchievementService.Delete)
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
