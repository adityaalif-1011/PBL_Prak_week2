package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/repository"
	"api-students/app/service"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
	"api-students/route"

	_ "api-students/docs" // swagger (docs di-generate oleh `go tool swag init`)
)

const minSecretLength = 32

// @title API Students
// @version 1.0
// @description API praktikum backend: students, courses, enrollments, reviews, dan notifikasi.
// @BasePath /api/v1
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Masukkan access token dengan format `Bearer <token>`.
func main() {
	config.LoadEnv()
	logger := config.NewLogger()

	role := flag.String("role", "all", "proses yang dijalankan: api, worker, atau all")
	flag.Parse()

	// Cek JWT_SECRET
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < minSecretLength {
		logger.Error("JWT_SECRET tidak diisi atau terlalu pendek",
			slog.Int("minimal_karakter", minSecretLength))
		os.Exit(1)
	}

	// Database
	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// JWT Manager
	jwtManager := helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	// Repository yang dipakai semua role
	txManager := repository.NewTxManager(pool)
	userRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewTokenRepository(pool)
	roleRepo := repository.NewRoleRepository(pool)
	studentRepo := repository.NewStudentRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	notifRepo := repository.NewNotificationRepository(pool)

	// Worker (butuh Redis)
	var workerShutdown func(context.Context) error
	if *role == "worker" || *role == "all" {
		workerShutdown, err = startWorker(context.Background(), logger, pool)
		if err != nil {
			logger.Error("gagal memulai worker", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	// API (butuh MongoDB + JWT + permissions)
	var apiShutdown func(context.Context) error
	if *role == "api" || *role == "all" {
		apiShutdown, err = startAPI(context.Background(), logger, pool,
			jwtManager, jwtSecret, userRepo, tokenRepo, roleRepo,
			studentRepo, txManager, outboxRepo, notifRepo)
		if err != nil {
			logger.Error("gagal memulai api", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	logger.Info("proses berjalan", slog.String("role", *role))

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Matikan HTTP dulu, lalu worker
	if apiShutdown != nil {
		if err := apiShutdown(shutdownCtx); err != nil {
			logger.Error("api shutdown error", slog.String("error", err.Error()))
		}
	}
	if workerShutdown != nil {
		if err := workerShutdown(shutdownCtx); err != nil {
			logger.Error("worker shutdown error", slog.String("error", err.Error()))
		}
	}

	logger.Info("shutdown complete")
}

// startAPI menyiapkan MongoDB, services, routes, dan HTTP server.
func startAPI(
	ctx context.Context,
	logger *slog.Logger,
	pool *pgxpool.Pool,
	jwtManager *helper.JWTManager,
	jwtSecret string,
	userRepo repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	roleRepo *repository.RoleRepository,
	studentRepo repository.StudentRepository,
	txManager *repository.TxManager,
	outboxRepo repository.OutboxRepository,
	notifRepo repository.NotificationRepository,
) (func(context.Context) error, error) {
	mongoClient, err := database.NewMongo(ctx)
	if err != nil {
		return nil, err
	}
	mongoDB := database.MongoDB(mongoClient)

	reviewRepo := repository.NewReviewRepository(mongoDB)
	portfolioRepo := repository.NewPortfolioRepository(mongoDB)

	if err := reviewRepo.EnsureIndexes(ctx); err != nil {
		return nil, err
	}
	if err := portfolioRepo.EnsureIndexes(ctx); err != nil {
		return nil, err
	}

	// Load permissions
	rawPermissions, err := roleRepo.LoadPermissions(ctx)
	if err != nil {
		return nil, err
	}
	permissions := helper.NewPermissionSet(rawPermissions)
	logger.Info("permission dimuat", slog.Any("roles", permissions.KnownRoles()))

	// Repositories lain
	achievementRepo := repository.NewAchievementRepository(pool)
	courseRepo := repository.NewCourseRepository(pool)
	supervisorRepo := repository.NewSupervisorRepository(pool)

	// Asynq client & inspector untuk monitor queue
	asynqClient := asynq.NewClient(config.RedisOpt())
	inspector := config.NewInspector()

	// Services
	studentService := service.NewStudentService(studentRepo, permissions, txManager)
	achievementService := service.NewAchievementService(achievementRepo, studentRepo)
	userService := service.NewUserService(userRepo)
	courseService := service.NewCourseService(courseRepo, txManager, outboxRepo)
	supervisorService := service.NewSupervisorService(supervisorRepo, studentRepo, txManager)
	reviewService := service.NewReviewService(reviewRepo, courseRepo)
	portfolioService := service.NewPortfolioService(portfolioRepo, studentRepo)
	notificationService := service.NewNotificationService(notifRepo, outboxRepo, inspector)
	authService := service.NewAuthService(
		userRepo,
		tokenRepo,
		jwtManager,
		time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7))*24*time.Hour,
	)

	// App
	app := config.NewApp(pool, route.Dependencies{
		Pool:                pool,
		Mongo:               mongoClient,
		JWT:                 jwtManager,
		Permissions:         permissions,
		StudentService:      studentService,
		AchievementService:  achievementService,
		AuthService:         authService,
		UserService:         userService,
		CourseService:       courseService,
		SupervisorService:   supervisorService,
		ReviewService:       reviewService,
		PortfolioService:    portfolioService,
		NotificationService: notificationService,
		EnableSwagger:       config.GetEnv("SWAGGER_ENABLED", "false") == "true",
	})

	port := config.GetEnv("APP_PORT", "3000")
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server stopped", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("api running", slog.String("port", port))

	shutdown := func(shutdownCtx context.Context) error {
		if err := mongoClient.Disconnect(shutdownCtx); err != nil {
			logger.Error("gagal menutup mongodb", slog.String("error", err.Error()))
		}
		asynqClient.Close()
		return app.ShutdownWithContext(shutdownCtx)
	}
	return shutdown, nil
}
