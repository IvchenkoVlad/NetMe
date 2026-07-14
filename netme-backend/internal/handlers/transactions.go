package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
)

// TxnRepo is the subset of PlaidRepository used by transaction handlers.
type TxnRepo interface {
	GetTransactionsByUserID(userID, accountID string, limit, offset int) ([]*models.Transaction, error)
	GetTransactionByID(userID, id string) (*models.Transaction, error)
	PatchTransactionCategory(userID, txnID, categoryID string) (*models.Transaction, error)
}

type TransactionsHandler struct {
	repo TxnRepo
}

func NewTransactionsHandler(repo TxnRepo) *TransactionsHandler {
	return &TransactionsHandler{repo: repo}
}

func RegisterTransactionRoutes(r *gin.RouterGroup, repo TxnRepo) {
	h := NewTransactionsHandler(repo)
	txns := r.Group("/transactions")
	{
		txns.GET("", h.ListTransactions)
		txns.GET("/:id", h.GetTransaction)
		txns.PATCH("/:id", h.PatchTransaction)
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

	txns, err := h.repo.GetTransactionsByUserID(uid, accountID, limit, offset)
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

func (h *TransactionsHandler) GetTransaction(c *gin.Context) {
	userID, _ := c.Get("user_id")
	txn, err := h.repo.GetTransactionByID(userID.(string), c.Param("id"))
	if err == sql.ErrNoRows || txn == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "transaction not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database_error", Message: "failed to load transaction"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transaction": txn})
}

func (h *TransactionsHandler) PatchTransaction(c *gin.Context) {
	var req struct {
		CategoryID string `json:"category_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid_request", Message: "category_id is required"})
		return
	}
	userID, _ := c.Get("user_id")
	txn, err := h.repo.PatchTransactionCategory(userID.(string), c.Param("id"), req.CategoryID)
	if err == sql.ErrNoRows || txn == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "transaction not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database_error", Message: "failed to update transaction"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transaction": txn})
}
