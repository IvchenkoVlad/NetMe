package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
)

type AccountsHandler struct {
	db *sql.DB
}

func NewAccountsHandler(db *sql.DB) *AccountsHandler {
	return &AccountsHandler{db: db}
}

func RegisterAccountRoutes(r *gin.RouterGroup, db *sql.DB) {
	NewAccountsHandler(db).RegisterRoutes(r)
}

func (h *AccountsHandler) RegisterRoutes(r *gin.RouterGroup) {
	accounts := r.Group("/accounts")
	{
		accounts.GET("", h.ListAccounts)
		accounts.GET("/:id", h.GetAccount)
	}
}

func (h *AccountsHandler) ListAccounts(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Accounts endpoint not yet implemented",
	})
}

func (h *AccountsHandler) GetAccount(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Accounts endpoint not yet implemented",
	})
}
