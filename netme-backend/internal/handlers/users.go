package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

type UsersHandler struct {
	userRepo repositories.UserRepo
	plaidSvc *services.PlaidService
}

func NewUsersHandler(userRepo repositories.UserRepo, plaidSvc *services.PlaidService) *UsersHandler {
	return &UsersHandler{userRepo: userRepo, plaidSvc: plaidSvc}
}

func (h *UsersHandler) GetMe(c *gin.Context) {
	user, err := h.userRepo.GetUserByID(uid(c))
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("user_not_found", "User not found"))
		return
	}
	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

func (h *UsersHandler) DeleteMe(c *gin.Context) {
	userID := uid(c)

	// Revoke all Plaid connections before removing the user from the database.
	// Required for CCPA/GDPR compliance and Plaid developer agreement.
	if h.plaidSvc != nil {
		h.plaidSvc.RevokeAllItems(c.Request.Context(), userID)
	}

	if err := h.userRepo.DeleteUser(userID); err != nil {
		dbErr(c, "Failed to delete account")
		return
	}
	c.Status(http.StatusNoContent)
}
