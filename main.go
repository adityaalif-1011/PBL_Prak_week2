package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
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

	api.Get("/students", GetStudents)
	api.Get("/students/:id", GetStudent)
	api.Post("/students", CreateStudent)
	api.Put("/students/:id", UpdateStudent)
	api.Patch("/students/:id", PatchStudent)
	api.Delete("/students/:id", DeleteStudent)

	log.Fatal(app.Listen(":3000"))
}
