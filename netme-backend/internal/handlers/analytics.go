package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type AnalyticsHandler struct {
	plaid  *repositories.PlaidRepository
	budget *repositories.BudgetRepository
}

func NewAnalyticsHandler(plaid *repositories.PlaidRepository, budget *repositories.BudgetRepository) *AnalyticsHandler {
	return &AnalyticsHandler{plaid: plaid, budget: budget}
}

func RegisterAnalyticsRoutes(r *gin.RouterGroup, plaid *repositories.PlaidRepository, budget *repositories.BudgetRepository) {
	h := NewAnalyticsHandler(plaid, budget)
	r.GET("/analytics/overview", h.Overview)
}

// GET /v1/analytics/overview
// Returns net worth, 6-month spending history, and top 5 spending categories for current month.
func (h *AnalyticsHandler) Overview(c *gin.Context) {
	userID := uid(c)
	month := time.Now().Format("2006-01")

	nw, err := h.plaid.GetNetWorth(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "db_error", Message: err.Error()})
		return
	}

	history, err := h.budget.GetMonthlyHistory(userID, 6)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "db_error", Message: err.Error()})
		return
	}
	if history == nil {
		history = []models.MonthlyTotal{}
	}

	topCats, err := h.budget.GetTopCategories(userID, month, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "db_error", Message: err.Error()})
		return
	}
	if topCats == nil {
		topCats = []models.TopCategory{}
	}

	c.JSON(http.StatusOK, models.AnalyticsOverview{
		NetWorth:      *nw,
		MonthlyTotals: history,
		TopCategories: topCats,
	})
}
