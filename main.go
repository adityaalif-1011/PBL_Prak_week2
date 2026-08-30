package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"api-students/app/database"
	"api-students/app/model"
	"api-students/app/repository"
	"api-students/config"
)

func main() {
	// Load environment variables
	config.LoadEnv()

	// Connect to database
	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Init Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnsupportedMediaType).JSON(model.ResponseEnvelope{
				Success: false,
				Message: "Content-Type must be application/json",
			})
		},
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())

	// Init Repository & Handler
	studentRepo := repository.NewStudentRepository(pool)
	studentHandler := NewStudentHandler(studentRepo)

	// API Routes
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(model.ResponseEnvelope{
				Success: false,
				Message: "database is unavailable",
			})
		}

		return c.Status(fiber.StatusOK).JSON(model.ResponseEnvelope{
			Success: true,
			Message: "server and database are running",
		})
	})

	// Student routes - using StudentHandler
	api.Get("/students", studentHandler.List)
	api.Get("/students/:id", studentHandler.Get)
	api.Post("/students", studentHandler.Create)
	api.Put("/students/:id", studentHandler.Replace)
	api.Patch("/students/:id", studentHandler.Patch)
	api.Delete("/students/:id", studentHandler.Delete)

	// Start server
	port := config.GetEnv("APP_PORT", "3000")
	log.Printf("server running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
