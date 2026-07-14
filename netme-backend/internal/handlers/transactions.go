package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
)

// TxnRepo is the subset of PlaidRepository used by transaction handlers.
type TxnRepo interface {
	GetTransactionsByUserID(userID, accountID, month string, limit, offset int) ([]*models.Transaction, error)
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
	month, ok := parseMonth(c)
	if !ok {
		return
	}

	limit, offset := 50, 0
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

	txns, err := h.repo.GetTransactionsByUserID(uid(c), accountID, month, limit, offset)
	if err != nil {
		dbErr(c, "failed to load transactions")
		return
	}
	if txns == nil {
		txns = []*models.Transaction{}
	}
	c.JSON(http.StatusOK, gin.H{"transactions": txns})
}

func (h *TransactionsHandler) GetTransaction(c *gin.Context) {
	txn, err := h.repo.GetTransactionByID(uid(c), c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) || txn == nil {
		c.JSON(http.StatusNotFound, errResp("not_found", "transaction not found"))
		return
	}
	if err != nil {
		dbErr(c, "failed to load transaction")
		return
	}
	c.JSON(http.StatusOK, gin.H{"transaction": txn})
}

func (h *TransactionsHandler) PatchTransaction(c *gin.Context) {
	var req struct {
		CategoryID string `json:"category_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", "category_id is required"))
		return
	}
	txn, err := h.repo.PatchTransactionCategory(uid(c), c.Param("id"), req.CategoryID)
	if errors.Is(err, sql.ErrNoRows) || txn == nil {
		c.JSON(http.StatusNotFound, errResp("not_found", "transaction not found"))
		return
	}
	if err != nil {
		dbErr(c, "failed to update transaction")
		return
	}
	c.JSON(http.StatusOK, gin.H{"transaction": txn})
}
