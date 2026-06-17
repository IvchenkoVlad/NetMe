package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
)

type TransactionsHandler struct {
	db *sql.DB
}

func NewTransactionsHandler(db *sql.DB) *TransactionsHandler {
	return &TransactionsHandler{db: db}
}

func RegisterTransactionRoutes(r *gin.RouterGroup, db *sql.DB) {
	NewTransactionsHandler(db).RegisterRoutes(r)
}

func (h *TransactionsHandler) RegisterRoutes(r *gin.RouterGroup) {
	txns := r.Group("/transactions")
	{
		txns.GET("", h.ListTransactions)
		txns.GET("/:id", h.GetTransaction)
	}
}

func (h *TransactionsHandler) ListTransactions(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Transactions endpoint not yet implemented",
	})
}

func (h *TransactionsHandler) GetTransaction(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Transactions endpoint not yet implemented",
	})
}
