package service

import (
	"encoding/csv"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

const (
	defaultUserLimit = 10
	maxUserLimit     = 100
)

// UserService - handler untuk resource /users.
type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// List - GET /api/v1/users (cursor pagination, tanpa OFFSET).
// @Summary Daftar user (cursor pagination)
// @Description Mengambil daftar user dengan cursor pagination. Mendukung content negotiation: kirim header Accept: text/csv untuk mengunduh CSV, selain itu JSON. Butuh permission user:list.
// @Tags users
// @Accept json
// @Produce json
// @Param limit query int false "Jumlah data per halaman (default 10, maks 100)" default(10)
// @Param cursor query string false "Cursor halaman berikutnya (dari meta.next_cursor)"
// @Param search query string false "Pencarian pada username/email"
// @Success 200 {object} model.ResponseEnvelope{data=[]model.User,meta=model.CursorMeta}
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 403 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /users [get]
func (s *UserService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// Negotiate dipanggil SEBELUM query supaya format sudah pasti.
	format := helper.Negotiate(c)

	q, err := parseUserQuery(c)
	if err != nil {
		return err
	}

	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.InternalError(err)
	}

	// Repository mengambil limit+1 baris untuk mendeteksi halaman berikutnya.
	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit]
	}

	if format == helper.FormatCSV {
		return s.writeUsersCSV(c, rows)
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessList(c, "users retrieved successfully", rows, meta)
}

// parseUserQuery membaca limit, cursor, dan search dari query string.
func parseUserQuery(c *fiber.Ctx) (model.UserQuery, error) {
	limit, _ := strconv.Atoi(c.Query("limit", strconv.Itoa(defaultUserLimit)))
	if limit < 1 {
		limit = defaultUserLimit
	}
	if limit > maxUserLimit {
		limit = maxUserLimit
	}

	q := model.UserQuery{
		Limit:    limit,
		CursorAt: time.Now().UTC(),
		CursorID: math.MaxInt,
		Search:   strings.TrimSpace(c.Query("search")),
	}

	if cursor := c.Query("cursor"); cursor != "" {
		at, id, err := helper.DecodeCursor(cursor)
		if err != nil {
			return model.UserQuery{}, helper.NewError(
				fiber.StatusBadRequest, "invalid_cursor", "cursor tidak valid")
		}
		q.CursorAt = at
		q.CursorID = id
	}

	return q, nil
}

// writeUsersCSV - kirim daftar user sebagai file CSV.
func (s *UserService) writeUsersCSV(c *fiber.Ctx, rows []model.User) error {
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="users.csv"`)

	if err := s.WriteUsersCSV(c.Response().BodyWriter(), rows); err != nil {
		return helper.InternalError(err)
	}
	return nil
}

// WriteUsersCSV menulis baris user ke writer dalam format CSV.
func (s *UserService) WriteUsersCSV(w io.Writer, rows []model.User) error {
	writer := csv.NewWriter(w)

	if err := writer.Write([]string{"id", "username", "email", "role", "is_active", "created_at"}); err != nil {
		return err
	}

	for _, u := range rows {
		record := []string{
			strconv.Itoa(u.ID),
			u.Username,
			u.Email,
			u.Role,
			strconv.FormatBool(u.IsActive),
			u.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}
