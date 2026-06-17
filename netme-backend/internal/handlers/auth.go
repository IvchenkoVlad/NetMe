package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

type AuthHandler struct {
	authRepo        *repositories.AuthRepository
	jwtService      *services.JWTService
	passwordService *services.PasswordService
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{
		authRepo:        repositories.NewAuthRepository(db),
		jwtService:      services.NewJWTService(),
		passwordService: services.NewPasswordService(),
	}
}

func RegisterAuthRoutes(router *gin.RouterGroup, db *sql.DB) {
	handler := NewAuthHandler(db)
	auth := router.Group("/auth")
	{
		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
		auth.POST("/refresh", handler.Refresh)
		auth.POST("/logout", handler.Logout)
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	existingUser, _ := h.authRepo.GetUserByEmail(req.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error:   "user_exists",
			Message: "User with this email already exists",
		})
		return
	}

	passwordHash, err := h.passwordService.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "hash_error",
			Message: "Failed to hash password",
		})
		return
	}

	user, err := h.authRepo.CreateUser(req.Email, passwordHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "creation_error",
			Message: "Failed to create user",
		})
		return
	}

	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token_error",
			Message: "Failed to generate token",
		})
		return
	}

	refreshTokenString, err := h.jwtService.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token_error",
			Message: "Failed to generate refresh token",
		})
		return
	}
	refreshToken, err := h.authRepo.CreateRefreshToken(user.ID, refreshTokenString, time.Now().Add(7*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token_error",
			Message: "Failed to store refresh token",
		})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusCreated, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    900, // 15 minutes
		User:         user,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	user, err := h.authRepo.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_credentials",
			Message: "Invalid email or password",
		})
		return
	}

	if user.PasswordHash == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_credentials",
			Message: "User has no password. Please use social login.",
		})
		return
	}

	if err := h.passwordService.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_credentials",
			Message: "Invalid email or password",
		})
		return
	}

	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token_error",
			Message: "Failed to generate token",
		})
		return
	}

	refreshTokenString, err := h.jwtService.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token_error",
			Message: "Failed to generate refresh token",
		})
		return
	}
	refreshToken, err := h.authRepo.CreateRefreshToken(user.ID, refreshTokenString, time.Now().Add(7*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token_error",
			Message: "Failed to store refresh token",
		})
		return
	}

	if err := h.authRepo.UpdateLastLogin(user.ID); err != nil {
		// Log but don't fail
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    900, // 15 minutes
		User:         user,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	valid, err := h.authRepo.IsRefreshTokenValid(req.RefreshToken)
	if !valid || err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_token",
			Message: "Refresh token is invalid or expired",
		})
		return
	}

	refreshTokenRecord, err := h.authRepo.GetRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_token",
			Message: "Refresh token not found",
		})
		return
	}

	user, err := h.authRepo.GetUserByID(refreshTokenRecord.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "user_not_found",
			Message: "User not found",
		})
		return
	}

	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token_error",
			Message: "Failed to generate token",
		})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: req.RefreshToken,
		ExpiresIn:    900, // 15 minutes
		User:         user,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_token",
			Message: "Authorization header is required",
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_format",
			Message: "Invalid authorization header format",
		})
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if err := h.authRepo.RevokeRefreshToken(req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "revoke_error",
			Message: "Failed to revoke token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}
