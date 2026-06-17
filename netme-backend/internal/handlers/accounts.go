package handlers

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func RegisterAccountRoutes(router *gin.RouterGroup, db *sql.DB) {
	accounts := router.Group("/accounts")
	{
		accounts.GET("", ListAccountsHandler(db))
		accounts.GET("/:id", GetAccountHandler(db))
	}
}

func ListAccountsHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement list accounts
		c.JSON(200, gin.H{"message": "list accounts endpoint"})
	}
}

func GetAccountHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement get account
		c.JSON(200, gin.H{"message": "get account endpoint"})
	}
}
