package handlers

import (
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
)

var reMonth = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// uid extracts the authenticated user ID set by AuthMiddleware.
// Panics only in programmer error (middleware bypassed); safe to call in all protected handlers.
func uid(c *gin.Context) string {
	v, _ := c.Get("user_id")
	return v.(string)
}

func currentMonth() string {
	return time.Now().Format("2006-01")
}

// parseMonth validates the optional ?month=YYYY-MM query param.
// Returns the current month if the param is absent, or writes a 400 and returns false on bad input.
func parseMonth(c *gin.Context) (string, bool) {
	m := c.Query("month")
	if m == "" {
		return currentMonth(), true
	}
	if !reMonth.MatchString(m) {
		c.JSON(http.StatusBadRequest, errResp("invalid_month", "month must be in YYYY-MM format (e.g. 2026-07)"))
		return "", false
	}
	return m, true
}

// errResp builds a consistent error response body.
func errResp(code, msg string) models.ErrorResponse {
	return models.ErrorResponse{Error: code, Message: msg}
}

// dbErr writes a 500 database error response. Convenience wrapper used in every handler.
func dbErr(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, errResp("db_error", msg))
}
