package handlers

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func RegisterAnalyticsRoutes(router *gin.RouterGroup, db *sql.DB) {
	analytics := router.Group("/analytics")
	{
		analytics.GET("/spending", SpendingHandler(db))
		analytics.GET("/net-worth", NetWorthHandler(db))
		analytics.GET("/categories", CategoriesHandler(db))
	}
}

func SpendingHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement spending analytics
		c.JSON(200, gin.H{"message": "spending analytics endpoint"})
	}
}

func NetWorthHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement net worth analytics
		c.JSON(200, gin.H{"message": "net worth analytics endpoint"})
	}
}

func CategoriesHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement categories analytics
		c.JSON(200, gin.H{"message": "categories analytics endpoint"})
	}
}
