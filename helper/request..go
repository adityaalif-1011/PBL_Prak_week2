package helper

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

// RequestContext memberi timeout 5 detik
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

// ParamID membaca dan validasi ID
func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// Allowed sort columns (whitelist)
var allowedSort = map[string]bool{
	"id":         true,
	"nim":        true,
	"name":       true,
	"grade":      true,
	"is_active":  true,
	"created_at": true,
}

// ParseListQuery membaca query string
func ParseListQuery(c *fiber.Ctx) model.ListQuery {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	q := model.ListQuery{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search", ""),
		Sort:   c.Query("sort", "id"),
		Order:  c.Query("order", "asc"),
	}

	if !allowedSort[q.Sort] {
		q.Sort = "id"
	}

	if q.Order != "desc" {
		q.Order = "asc"
	}

	if isActive := c.Query("is_active"); isActive != "" {
		if val, err := strconv.ParseBool(isActive); err == nil {
			q.IsActive = &val
		}
	}

	if minGrade := c.Query("min_grade"); minGrade != "" {
		q.MinGrade, _ = strconv.Atoi(minGrade)
	}
	if maxGrade := c.Query("max_grade"); maxGrade != "" {
		q.MaxGrade, _ = strconv.Atoi(maxGrade)
	}

	return q
}
