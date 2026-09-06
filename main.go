package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-students/app/repository"
	"api-students/app/service"
	"api-students/config"
	"api-students/database"
)

func main() {
	// 1. Load env
	config.LoadEnv()

	// 2. Logger
	logger := config.NewLogger()

	// 3. Database
	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// 4. Repository -> Service (perakitan dari dalam ke luar)
	studentRepo := repository.NewStudentRepository(pool)
	studentService := service.NewStudentService(studentRepo)

	// 5. App (perakitan aplikasi)
	app := config.NewApp(pool, studentService)

	// 6. Run server
	port := config.GetEnv("APP_PORT", "3000")
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server stopped", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("server running", slog.String("port", port))

	// 7. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("server stopped gracefully")
}
