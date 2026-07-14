package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

type UsersHandler struct {
	userRepo  repositories.UserRepo
	plaidSvc  *services.PlaidService
}

func NewUsersHandler(userRepo repositories.UserRepo, plaidSvc *services.PlaidService) *UsersHandler {
	return &UsersHandler{userRepo: userRepo, plaidSvc: plaidSvc}
}

func (h *UsersHandler) GetMe(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Not authenticated",
		})
		return
	}
	user, err := h.userRepo.GetUserByID(userIDVal.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "user_not_found",
			Message: "User not found",
		})
		return
	}
	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

func (h *UsersHandler) DeleteMe(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Not authenticated",
		})
		return
	}

	userID := userIDVal.(string)

	// Revoke all Plaid connections before removing the user from the database.
	// Required for CCPA/GDPR compliance and Plaid developer agreement.
	if h.plaidSvc != nil {
		h.plaidSvc.RevokeAllItems(c.Request.Context(), userID)
	}

	if err := h.userRepo.DeleteUser(userID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "delete_error",
			Message: "Failed to delete account",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
