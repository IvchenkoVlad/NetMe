package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type UsersHandler struct {
	authRepo *repositories.AuthRepository
}

func NewUsersHandler(db *sql.DB) *UsersHandler {
	return &UsersHandler{authRepo: repositories.NewAuthRepository(db)}
}

func (h *UsersHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.authRepo.GetUserByID(userID.(string))
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
