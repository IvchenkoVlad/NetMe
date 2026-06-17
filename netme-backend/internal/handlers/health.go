package handlers

import (
	"github.com/gin-gonic/gin"
)

// HealthHandler returns a simple health check response
func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Hello from NetMe Backend!",
		})
	}
}

// HelloHandler returns a greeting message
func HelloHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.DefaultQuery("name", "World")
		c.JSON(200, gin.H{
			"message": "Hello, " + name + "!",
			"backend": "NetMe API v1",
			"status":  "running",
		})
	}
}
