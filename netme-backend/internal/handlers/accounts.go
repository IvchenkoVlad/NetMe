package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type AccountsHandler struct {
	repo repositories.AccountLister
}

func NewAccountsHandler(repo repositories.AccountLister) *AccountsHandler {
	return &AccountsHandler{repo: repo}
}

func RegisterAccountRoutes(r *gin.RouterGroup, repo repositories.AccountLister) {
	h := NewAccountsHandler(repo)
	r.Group("/accounts").GET("", h.ListAccounts)
}

func (h *AccountsHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.repo.GetAccountsByUserID(uid(c))
	if err != nil {
		dbErr(c, "failed to load accounts")
		return
	}
	if accounts == nil {
		accounts = []*models.Account{}
	}
	c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}
