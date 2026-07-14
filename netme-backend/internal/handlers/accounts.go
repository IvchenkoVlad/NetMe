package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)


type AccountsHandler struct {
	plaidRepo *repositories.PlaidRepository
}

func NewAccountsHandler(repo *repositories.PlaidRepository) *AccountsHandler {
	return &AccountsHandler{plaidRepo: repo}
}

func RegisterAccountRoutes(r *gin.RouterGroup, repo *repositories.PlaidRepository) {
	NewAccountsHandler(repo).RegisterRoutes(r)
}

func (h *AccountsHandler) RegisterRoutes(r *gin.RouterGroup) {
	accounts := r.Group("/accounts")
	{
		accounts.GET("", h.ListAccounts)
	}
}

func (h *AccountsHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.plaidRepo.GetAccountsByUserID(uid(c))
	if err != nil {
		dbErr(c, "failed to load accounts")
		return
	}
	if accounts == nil {
		accounts = []*models.Account{}
	}
	c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}
