package helper

import (
	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

// Success mengirim response 2xx dengan data
func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.ResponseEnvelope{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessList mengirim response daftar dengan meta
func SuccessList(c *fiber.Ctx, message string, data any, meta *model.MetaData) error {
	return c.Status(fiber.StatusOK).JSON(model.ResponseEnvelope{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Created mengirim 201 dengan Location header
func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.ResponseEnvelope{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// NoContent mengirim 204 tanpa body
func NoContent(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNoContent).JSON(nil)
}

// Fail mengirim response error
func Fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.ResponseEnvelope{
		Success: false,
		Message: message,
	})
}

// FailValidation mengirim 422 dengan detail error
func FailValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.ResponseEnvelope{
		Success: false,
		Message: "validation failed",
		Errors:  errs,
	})
}
