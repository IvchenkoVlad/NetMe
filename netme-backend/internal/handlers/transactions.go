package handlers

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func RegisterTransactionRoutes(router *gin.RouterGroup, db *sql.DB) {
	transactions := router.Group("/transactions")
	{
		transactions.GET("", ListTransactionsHandler(db))
		transactions.GET("/:id", GetTransactionHandler(db))
		transactions.GET("/search", SearchTransactionsHandler(db))
	}
}

func ListTransactionsHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement list transactions
		c.JSON(200, gin.H{"message": "list transactions endpoint"})
	}
}

func GetTransactionHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement get transaction
		c.JSON(200, gin.H{"message": "get transaction endpoint"})
	}
}

func SearchTransactionsHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement search transactions
		c.JSON(200, gin.H{"message": "search transactions endpoint"})
	}
}
