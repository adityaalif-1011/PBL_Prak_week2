package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

const refreshTokenBytes = 32

type AuthService struct {
	users      repository.UserRepository
	tokens     *repository.TokenRepository
	jwt        *helper.JWTManager
	refreshTTL time.Duration
}

func NewAuthService(
	users repository.UserRepository,
	tokens *repository.TokenRepository,
	jwt *helper.JWTManager,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:      users,
		tokens:     tokens,
		jwt:        jwt,
		refreshTTL: refreshTTL,
	}
}

// Register - POST /auth/register
// @Summary Mendaftarkan user baru
// @Description Mendaftarkan akun baru. Role selalu ditentukan server (user) agar tidak terjadi mass assignment.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body model.RegisterRequest true "Data pendaftaran"
// @Success 201 {object} model.ResponseEnvelope
// @Failure 400 {object} helper.ErrorResponse
// @Failure 409 {object} helper.ErrorResponse
// @Failure 422 {object} helper.ErrorResponse
// @Router /auth/register [post]
func (s *AuthService) Register(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "body harus berupa JSON yang valid")
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	// Hash password
	hashed, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.WrapError(fiber.StatusInternalServerError, "internal_error", "gagal memproses password", err)
	}

	// Role selalu ditentukan server
	created, err := s.users.Create(ctx, model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashed,
		Role:     "user",
		IsActive: true,
	})
	if err != nil {
		if errors.Is(err, repository.ErrUserDuplicate) {
			return helper.NewError(fiber.StatusConflict, "conflict", "username atau email sudah dipakai")
		}
		return helper.WrapError(fiber.StatusInternalServerError, "internal_error", "gagal mendaftarkan user", err)
	}

	return helper.Created(c, "pendaftaran berhasil", created,
		"/api/v1/users/"+strconv.Itoa(created.ID))
}

// Login - POST /auth/login
// @Summary Login dan ambil pasangan token
// @Description Mengautentikasi username & password, lalu mengembalikan access token dan refresh token.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body model.LoginRequest true "Kredensial login"
// @Success 200 {object} model.ResponseEnvelope
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 403 {object} helper.ErrorResponse
// @Failure 422 {object} helper.ErrorResponse
// @Router /auth/login [post]
func (s *AuthService) Login(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	user, err := s.users.FindByUsername(ctx, strings.TrimSpace(req.Username))
	if err != nil {
		// PESAN SAMA PERSIS untuk cegah user enumeration
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "username atau password salah")
	}

	if !helper.VerifyPassword(user.Password, req.Password) {
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "username atau password salah")
	}

	if !user.IsActive {
		return helper.NewError(fiber.StatusForbidden, "account_disabled", "akun dinonaktifkan")
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return helper.WrapError(fiber.StatusInternalServerError, "internal_error", "gagal membuat token", err)
	}

	return helper.Success(c, fiber.StatusOK, "login berhasil", pair)
}

// Refresh - POST /auth/refresh
// @Summary Perbarui access token
// @Description Menukar refresh token yang masih berlaku dengan pasangan token baru.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body model.RefreshRequest true "Refresh token"
// @Success 200 {object} model.ResponseEnvelope
// @Failure 400 {object} helper.ErrorResponse
// @Failure 401 {object} helper.ErrorResponse
// @Failure 422 {object} helper.ErrorResponse
// @Router /auth/refresh [post]
func (s *AuthService) Refresh(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.ValidationFailed(errs)
	}

	hash := helper.SHA256Hex(strings.TrimSpace(req.RefreshToken))

	stored, err := s.tokens.FindActive(ctx, hash)
	if err != nil {
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized",
			"refresh token tidak valid atau sudah kedaluwarsa")
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil || !user.IsActive {
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "akun tidak dapat dipakai")
	}

	// ROTASI: cabut token lama, buat baru
	if err := s.tokens.Revoke(ctx, hash); err != nil {
		return helper.WrapError(fiber.StatusInternalServerError, "internal_error", "gagal memperbarui token", err)
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return helper.WrapError(fiber.StatusInternalServerError, "internal_error", "gagal membuat token", err)
	}

	return helper.Success(c, fiber.StatusOK, "token berhasil diperbarui", pair)
}

// Logout - POST /auth/logout
// @Summary Logout
// @Description Menarik refresh token (revoke berdasarkan hash) sehingga tidak bisa dipakai lagi. Walaupun body kosong/token tidak dikenal, request tetap dianggap berhasil.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body model.RefreshRequest true "Refresh token yang dicabut (opsional)"
// @Success 200 {object} model.ResponseEnvelope
// @Failure 400 {object} helper.ErrorResponse
// @Router /auth/logout [post]
func (s *AuthService) Logout(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewError(fiber.StatusBadRequest, "bad_request", "body harus berupa JSON yang valid")
	}

	if token := strings.TrimSpace(req.RefreshToken); token != "" {
		_ = s.tokens.Revoke(ctx, helper.SHA256Hex(token))
	}

	return helper.Success(c, fiber.StatusOK, "logout berhasil", nil)
}

// Me - GET /auth/me
// @Summary Profil user yang sedang login
// @Description Mengambil data user berdasarkan access token pada header Authorization.
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} model.ResponseEnvelope{data=model.User}
// @Failure 401 {object} helper.ErrorResponse
// @Security BearerAuth
// @Router /auth/me [get]
func (s *AuthService) Me(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	authUser, ok := helper.CurrentUser(c)
	if !ok {
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "belum terautentikasi")
	}

	user, err := s.users.FindByID(ctx, authUser.UserID)
	if err != nil {
		return helper.NewError(fiber.StatusUnauthorized, "unauthorized", "user tidak ditemukan")
	}

	return helper.Success(c, fiber.StatusOK, "profil berhasil diambil", user)
}

// issueTokenPair - buat access + refresh token
func (s *AuthService) issueTokenPair(ctx context.Context, user model.User) (model.TokenPair, error) {
	accessToken, err := s.jwt.GenerateAccess(user)
	if err != nil {
		return model.TokenPair{}, err
	}

	refreshToken, err := helper.RandomToken(refreshTokenBytes)
	if err != nil {
		return model.TokenPair{}, err
	}

	err = s.tokens.Save(ctx, model.RefreshToken{
		UserID:    user.ID,
		TokenHash: helper.SHA256Hex(refreshToken),
		ExpiresAt: time.Now().Add(s.refreshTTL),
	})
	if err != nil {
		return model.TokenPair{}, err
	}

	return model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwt.AccessTTL().Seconds()),
	}, nil
}
