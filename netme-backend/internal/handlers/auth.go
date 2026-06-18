package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

type AuthHandler struct {
	userRepo        repositories.UserRepo
	tokenRepo       repositories.TokenRepo
	jwtService      *services.JWTService
	passwordService *services.PasswordService
	googleVerifier  services.GoogleVerifier
}

func NewAuthHandler(userRepo repositories.UserRepo, tokenRepo repositories.TokenRepo, jwtSvc *services.JWTService, googleVerifier services.GoogleVerifier) *AuthHandler {
	return &AuthHandler{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		jwtService:      jwtSvc,
		passwordService: services.NewPasswordService(),
		googleVerifier:  googleVerifier,
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

	existingUser, _ := h.userRepo.GetUserByEmail(req.Email)
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

	user, err := h.userRepo.CreateUser(req.Email, passwordHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "creation_error",
			Message: "Failed to create user",
		})
		return
	}

	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate token"})
		return
	}

	refreshTokenString, err := h.jwtService.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate refresh token"})
		return
	}

	refreshToken, err := h.tokenRepo.CreateRefreshToken(user.ID, refreshTokenString, time.Now().Add(7*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to store refresh token"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusCreated, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    900,
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

	user, err := h.userRepo.GetUserByEmail(req.Email)
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
			Message: "No password set for this account",
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
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate token"})
		return
	}

	refreshTokenString, err := h.jwtService.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate refresh token"})
		return
	}

	refreshToken, err := h.tokenRepo.CreateRefreshToken(user.ID, refreshTokenString, time.Now().Add(7*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to store refresh token"})
		return
	}

	if err := h.userRepo.UpdateLastLogin(user.ID); err != nil {
		slog.Warn("failed to update last login", "user_id", user.ID, "error", err)
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    900,
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

	valid, err := h.tokenRepo.IsRefreshTokenValid(req.RefreshToken)
	if !valid || err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_token",
			Message: "Refresh token is invalid or expired",
		})
		return
	}

	refreshTokenRecord, err := h.tokenRepo.GetRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_token",
			Message: "Refresh token not found",
		})
		return
	}

	user, err := h.userRepo.GetUserByID(refreshTokenRecord.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "user_not_found",
			Message: "User not found",
		})
		return
	}

	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate token"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: req.RefreshToken,
		ExpiresIn:    900,
		User:         user,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	userID := c.GetString("user_id")
	if err := h.tokenRepo.RevokeRefreshToken(req.RefreshToken, userID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "revoke_error",
			Message: "Failed to revoke token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	var req struct {
		IDToken string `json:"id_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	googleID, email, err := h.googleVerifier.Validate(c.Request.Context(), req.IDToken, "")
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid_token", Message: "Failed to verify Google token"})
		return
	}

	user, err := h.userRepo.FindOrCreateGoogleUser(googleID, email)
	if err != nil {
		if errors.Is(err, repositories.ErrEmailTakenByOtherProvider) {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "email_conflict",
				Message: "An account with this email already exists. Please log in with your password.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "db_error", Message: "Failed to sign in with Google"})
		return
	}

	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate token"})
		return
	}

	refreshTokenString, err := h.jwtService.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate refresh token"})
		return
	}

	refreshToken, err := h.tokenRepo.CreateRefreshToken(user.ID, refreshTokenString, time.Now().Add(7*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to store refresh token"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    900,
		User:         user,
	})
}
