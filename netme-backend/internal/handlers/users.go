package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type UsersHandler struct {
	userRepo repositories.UserRepo
}

func NewUsersHandler(userRepo repositories.UserRepo) *UsersHandler {
	return &UsersHandler{userRepo: userRepo}
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
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Account deletion is not yet available",
	})
}
