package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type AnalyticsHandler struct {
	netWorth repositories.NetWorthReader
	budget   *repositories.BudgetRepository
}

func NewAnalyticsHandler(netWorth repositories.NetWorthReader, budget *repositories.BudgetRepository) *AnalyticsHandler {
	return &AnalyticsHandler{netWorth: netWorth, budget: budget}
}

func RegisterAnalyticsRoutes(r *gin.RouterGroup, netWorth repositories.NetWorthReader, budget *repositories.BudgetRepository) {
	h := NewAnalyticsHandler(netWorth, budget)
	r.GET("/analytics/overview", h.Overview)
}

// GET /v1/analytics/overview
// Returns net worth, 6-month spending history, and top 5 spending categories for current month.
func (h *AnalyticsHandler) Overview(c *gin.Context) {
	userID := uid(c)
	month := currentMonth()

	nw, err := h.netWorth.GetNetWorth(userID)
	if err != nil {
		dbErr(c, err.Error())
		return
	}

	history, err := h.budget.GetMonthlyHistory(userID, 6)
	if err != nil {
		dbErr(c, err.Error())
		return
	}
	if history == nil {
		history = []models.MonthlyTotal{}
	}

	topCats, err := h.budget.GetTopCategories(userID, month, 5)
	if err != nil {
		dbErr(c, err.Error())
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
