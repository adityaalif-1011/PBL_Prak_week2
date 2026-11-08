package config

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"

	"api-students/helper"
)

func TestErrorHandlerJSON(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: newErrorHandler()})

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("requestid", "req-123")
		if c.Path() == "/anything" {
			return helper.NewError(fiber.StatusNotFound, "not_found", "endpoint not found")
		}
		return c.Next()
	})
	app.Get("/forbidden", func(c *fiber.Ctx) error {
		return helper.NewError(fiber.StatusForbidden, "forbidden", "tidak boleh")
	})
	app.Get("/boom500", func(c *fiber.Ctx) error {
		return helper.InternalError(io.ErrUnexpectedEOF)
	})
	app.Get("/boom422", func(c *fiber.Ctx) error {
		return helper.ValidationFailed(map[string]string{"username": "wajib diisi"})
	})

	cases := []struct {
		path   string
		status int
		code   string
	}{
		{"/anything", fiber.StatusNotFound, "not_found"},
		{"/forbidden", fiber.StatusForbidden, "forbidden"},
		{"/boom500", fiber.StatusInternalServerError, "internal_error"},
		{"/boom422", fiber.StatusUnprocessableEntity, "validation_failed"},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(fiber.MethodGet, tc.path, nil)
		if err != nil {
			t.Fatalf("buat request gagal: %v", err)
		}

		resp, err := app.Test(req, 1000)
		if err != nil {
			t.Fatalf("test request gagal: %v", err)
		}
		if resp.StatusCode != tc.status {
			t.Errorf("%s: status got %d, want %d", tc.path, resp.StatusCode, tc.status)
		}

		body, _ := io.ReadAll(resp.Body)
		var payload helper.ErrorResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("%s: response bukan JSON valid: %s", tc.path, body)
		}
		if payload.Success {
			t.Errorf("%s: success harus false", tc.path)
		}
		if payload.Code != tc.code {
			t.Errorf("%s: code got %q, want %q", tc.path, payload.Code, tc.code)
		}
		if tc.path == "/anything" && payload.RequestID != "req-123" {
			t.Errorf("request_id got %q, want req-123", payload.RequestID)
		}
	}
}
