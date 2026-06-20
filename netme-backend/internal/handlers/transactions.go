package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type TransactionsHandler struct {
	plaidRepo *repositories.PlaidRepository
}

func NewTransactionsHandler(repo *repositories.PlaidRepository) *TransactionsHandler {
	return &TransactionsHandler{plaidRepo: repo}
}

func RegisterTransactionRoutes(r *gin.RouterGroup, repo *repositories.PlaidRepository) {
	NewTransactionsHandler(repo).RegisterRoutes(r)
}

func (h *TransactionsHandler) RegisterRoutes(r *gin.RouterGroup) {
	txns := r.Group("/transactions")
	{
		txns.GET("", h.ListTransactions)
	}
}

func (h *TransactionsHandler) ListTransactions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	limit := 50
	offset := 0
	accountID := c.Query("account_id")
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	txns, err := h.plaidRepo.GetTransactionsByUserID(uid, accountID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "failed to load transactions",
		})
		return
	}
	if txns == nil {
		txns = []*models.Transaction{}
	}
	c.JSON(http.StatusOK, gin.H{"transactions": txns})
}
