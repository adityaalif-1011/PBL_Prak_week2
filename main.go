package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"api-students/app/database"
	"api-students/config"
)

func main() {
	config.LoadEnv()

	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnsupportedMediaType).JSON(ResponseEnvelope{
				Success: false,
				Message: "Content-Type must be application/json",
			})
		},
	})

	app.Use(logger.New())
	app.Use(recover.New())

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(ResponseEnvelope{
				Success: false,
				Message: "database is unavailable",
			})
		}

		return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
			Success: true,
			Message: "server and database are running",
		})
	})

	api.Get("/students", GetStudents)
	api.Get("/students/:id", GetStudent)
	api.Post("/students", CreateStudent)
	api.Put("/students/:id", UpdateStudent)
	api.Patch("/students/:id", PatchStudent)
	api.Delete("/students/:id", DeleteStudent)

	port := config.GetEnv("APP_PORT", "3000")
	log.Printf("server running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
