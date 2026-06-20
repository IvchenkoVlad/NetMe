package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

type AuthHandler struct {
	authSvc *services.AuthService
}

func NewAuthHandler(authSvc *services.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", err.Error()))
		return
	}
	resp, err := h.authSvc.Register(req.Email, req.Password)
	if err != nil {
		c.JSON(authErrStatus(err), authErrBody(err))
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", err.Error()))
		return
	}
	resp, err := h.authSvc.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(authErrStatus(err), authErrBody(err))
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", err.Error()))
		return
	}
	resp, err := h.authSvc.Refresh(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errResp("invalid_token", "Refresh token is invalid or expired"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", err.Error()))
		return
	}
	if err := h.authSvc.Logout(req.RefreshToken, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, errResp("revoke_error", "Failed to revoke token"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	var req struct {
		IDToken string `json:"id_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", err.Error()))
		return
	}
	resp, err := h.authSvc.GoogleAuth(c.Request.Context(), req.IDToken)
	if err != nil {
		if errors.Is(err, repositories.ErrEmailTakenByOtherProvider) {
			c.JSON(http.StatusConflict, errResp("email_conflict", "An account with this email already exists. Please log in with your password."))
			return
		}
		c.JSON(http.StatusUnauthorized, errResp("invalid_token", "Failed to verify Google token"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

func errResp(code, msg string) models.ErrorResponse {
	return models.ErrorResponse{Error: code, Message: msg}
}

func authErrStatus(err error) int {
	switch {
	case errors.Is(err, services.ErrUserExists):
		return http.StatusConflict
	case errors.Is(err, services.ErrInvalidCredentials),
		errors.Is(err, services.ErrNoPassword),
		errors.Is(err, services.ErrInvalidToken):
		return http.StatusUnauthorized
	case errors.Is(err, services.ErrInvalidEmail):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func authErrBody(err error) models.ErrorResponse {
	switch {
	case errors.Is(err, services.ErrUserExists):
		return errResp("user_exists", "User with this email already exists")
	case errors.Is(err, services.ErrInvalidCredentials):
		return errResp("invalid_credentials", "Invalid email or password")
	case errors.Is(err, services.ErrNoPassword):
		return errResp("invalid_credentials", "No password set for this account")
	case errors.Is(err, services.ErrInvalidToken):
		return errResp("invalid_token", "Invalid or expired token")
	case errors.Is(err, services.ErrInvalidEmail):
		return errResp("invalid_request", "Invalid email address")
	default:
		return errResp("internal_error", "An unexpected error occurred")
	}
}
