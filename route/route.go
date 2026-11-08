package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"api-students/app/service"
	"api-students/helper"
	"api-students/middleware"
)

// Dependencies - kumpulan dependency
type Dependencies struct {
	Pool                *pgxpool.Pool
	Mongo               *mongo.Client
	JWT                 *helper.JWTManager
	Permissions         *helper.PermissionSet // ← TAMBAH
	StudentService      *service.StudentService
	AchievementService  *service.AchievementService
	AuthService         *service.AuthService
	UserService         *service.UserService
	CourseService       *service.CourseService
	SupervisorService   *service.SupervisorService
	ReviewService       *service.ReviewService
	PortfolioService    *service.PortfolioService
	NotificationService *service.NotificationService

	// EnableSwagger membuka Swagger UI di /swagger/ (false di produksi).
	EnableSwagger bool
}

func Register(app *fiber.App, deps Dependencies) {
	// ============================================================
	// SWAGGER UI (opsional, dikontrol env SWAGGER_ENABLED)
	// ============================================================
	if deps.EnableSwagger {
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	api := app.Group("/api/v1")

	// ============================================================
	// HEALTH (PUBLIK) - cek PostgreSQL DAN MongoDB
	// ============================================================
	api.Get("/health", healthCheck(deps.Pool, deps.Mongo))

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
	// STUDENTS (WAJIB LOGIN + PERMISSION)
	// ============================================================
	students := api.Group("/students",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT),
	)

	// Hak bisa diputuskan TANPA lihat data → middleware
	students.Get("/",
		middleware.RequirePermission(deps.Permissions, "student:list"),
		deps.StudentService.List)
	students.Post("/",
		middleware.RequirePermission(deps.Permissions, "student:create"),
		deps.StudentService.Create)
	students.Delete("/:id",
		middleware.RequirePermission(deps.Permissions, "student:delete"),
		deps.StudentService.Delete)

	// Hak BUTUH lihat data → service (ownership check)
	students.Get("/:id", deps.StudentService.Get)
	students.Put("/:id", deps.StudentService.Replace)
	students.Patch("/:id", deps.StudentService.Patch)

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

	// ============================================================
	// USERS (WAJIB LOGIN + PERMISSION) - cursor pagination & CSV
	// ============================================================
	users := api.Group("/users", middleware.RequireAuth(deps.JWT))
	users.Get("/",
		middleware.RequirePermission(deps.Permissions, "user:list"),
		deps.UserService.List)

	// ============================================================
	// COURSES & ENROLLMENTS (Modul 8 & 9)
	// ============================================================
	courses := api.Group("/courses", middleware.RequireAuth(deps.JWT))
	courses.Get("/", deps.CourseService.List)
	// /mine didaftarkan sebelum /:id agar tidak tertangkap sebagai :id.
	courses.Get("/mine", deps.CourseService.Mine)
	courses.Get("/:id", deps.CourseService.Get)
	courses.Post("/", middleware.RequirePermission(deps.Permissions, "student:create"), deps.CourseService.Create)
	courses.Put("/:id/quota", middleware.RequirePermission(deps.Permissions, "student:update:any"), deps.CourseService.UpdateQuota)

	// Reviews course (MongoDB)
	courses.Get("/:id/reviews", deps.ReviewService.ListByCourse)
	courses.Post("/:id/reviews", deps.ReviewService.Create)
	courses.Get("/:id/reviews/summary", deps.ReviewService.Summary)

	reviews := api.Group("/reviews", middleware.RequireAuth(deps.JWT))
	reviews.Post("/:id/replies", deps.ReviewService.AddReply)
	reviews.Post("/:id/helpful", deps.ReviewService.MarkHelpful)

	enrollments := api.Group("/enrollments", middleware.RequireAuth(deps.JWT))
	enrollments.Post("/", deps.CourseService.Enroll)
	enrollments.Post("/swap", deps.CourseService.Swap)
	enrollments.Delete("/:id", deps.CourseService.Drop)

	// ============================================================
	// SUPERVISORS / SUPERVISIONS (Modul 9)
	// ============================================================
	supervisors := api.Group("/supervisors", middleware.RequireAuth(deps.JWT))
	supervisors.Post("/", middleware.RequirePermission(deps.Permissions, "student:create"), deps.SupervisorService.Create)
	supervisors.Get("/:id", deps.SupervisorService.Get)
	supervisors.Patch("/:id", middleware.RequirePermission(deps.Permissions, "student:update:any"), deps.SupervisorService.Patch)
	supervisors.Post("/:id/assign", middleware.RequirePermission(deps.Permissions, "student:update:any"), deps.SupervisorService.Assign)

	supervisions := api.Group("/supervisions", middleware.RequireAuth(deps.JWT))
	supervisions.Post("/switch", middleware.RequirePermission(deps.Permissions, "student:update:any"), deps.SupervisorService.Switch)

	// ============================================================
	// PORTFOLIO (Modul 10 - MongoDB)
	// ============================================================
	students.Get("/:id/portfolio", deps.PortfolioService.Get)
	students.Put("/:id/portfolio", middleware.RequirePermission(deps.Permissions, "student:update:any"), deps.PortfolioService.Upsert)
	students.Post("/:id/portfolio/projects", middleware.RequirePermission(deps.Permissions, "student:update:any"), deps.PortfolioService.AddProject)
	students.Delete("/:id/portfolio/projects/:projectId", middleware.RequirePermission(deps.Permissions, "student:update:any"), deps.PortfolioService.DeleteProject)
	students.Get("/:id/portfolio/technologies", deps.PortfolioService.TopTechnologies)

	portfolios := api.Group("/portfolios", middleware.RequireAuth(deps.JWT))
	portfolios.Get("/stats/technologies", deps.PortfolioService.GlobalStats)

	// ============================================================
	// NOTIFICATIONS & QUEUE MONITOR (Modul 11)
	// ============================================================
	notif := api.Group("/notifications", middleware.RequireAuth(deps.JWT))
	notif.Get("/mine", deps.NotificationService.Mine)

	admin := api.Group("/admin", middleware.RequireAuth(deps.JWT))
	admin.Get("/queue",
		middleware.RequirePermission(deps.Permissions, "queue:monitor"),
		deps.NotificationService.Monitor)
}

func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.NewError(fiber.StatusServiceUnavailable, "service_unavailable", "postgresql is unavailable")
		}
		if err := mongoClient.Ping(ctx, readpref.Primary()); err != nil {
			return helper.NewError(fiber.StatusServiceUnavailable, "service_unavailable", "mongodb is unavailable")
		}
		return helper.Success(c, fiber.StatusOK, "server, postgresql, and mongodb are running", nil)
	}
}
