package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type BudgetHandler struct {
	repo *repositories.BudgetRepository
}

func NewBudgetHandler(repo *repositories.BudgetRepository) *BudgetHandler {
	return &BudgetHandler{repo: repo}
}

func RegisterBudgetRoutes(r *gin.RouterGroup, repo *repositories.BudgetRepository) {
	h := NewBudgetHandler(repo)
	r.GET("/budget/summary", h.GetSummary)
	r.GET("/budget/history", h.GetHistory)

	cats := r.Group("/categories")
	{
		cats.GET("", h.ListCategories)
		cats.POST("", h.CreateCategory)
		cats.PUT("/:id", h.UpdateCategory)
		cats.DELETE("/:id", h.DeleteCategory)
	}

	r.PUT("/budget/:category_id", h.SetBudget)
}

// GET /v1/budget/summary?month=2026-06
func (h *BudgetHandler) GetSummary(c *gin.Context) {
	month, ok := parseMonth(c)
	if !ok {
		return
	}
	summary, err := h.repo.BuildSummary(uid(c), month)
	if err != nil {
		dbErr(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GET /v1/budget/history?months=6
func (h *BudgetHandler) GetHistory(c *gin.Context) {
	months := 6
	if m := c.Query("months"); m != "" {
		if v, err := strconv.Atoi(m); err == nil && v > 0 && v <= 24 {
			months = v
		}
	}
	history, err := h.repo.GetMonthlyHistory(uid(c), months)
	if err != nil {
		dbErr(c, err.Error())
		return
	}
	if history == nil {
		history = []models.MonthlyTotal{}
	}
	c.JSON(http.StatusOK, gin.H{"history": history})
}

// GET /v1/categories
func (h *BudgetHandler) ListCategories(c *gin.Context) {
	userID := uid(c)
	if err := h.repo.EnsureCategories(userID); err != nil {
		dbErr(c, err.Error())
		return
	}
	cats, err := h.repo.GetCategories(userID)
	if err != nil {
		dbErr(c, err.Error())
		return
	}
	if cats == nil {
		cats = []*models.Category{}
	}
	c.JSON(http.StatusOK, gin.H{"categories": cats})
}

// POST /v1/categories
func (h *BudgetHandler) CreateCategory(c *gin.Context) {
	var req struct {
		Name     string   `json:"name" binding:"required"`
		Icon     string   `json:"icon"`
		Color    string   `json:"color"`
		IsIncome bool     `json:"is_income"`
		Plaid    []string `json:"plaid_primary_categories"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", err.Error()))
		return
	}
	if req.Icon == "" {
		req.Icon = "📦"
	}
	if req.Color == "" {
		req.Color = "#94a3b8"
	}
	if req.Plaid == nil {
		req.Plaid = []string{}
	}
	cat, err := h.repo.CreateCategory(uid(c), req.Name, req.Icon, req.Color, req.IsIncome, req.Plaid)
	if err != nil {
		dbErr(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, cat)
}

// PUT /v1/categories/:id
func (h *BudgetHandler) UpdateCategory(c *gin.Context) {
	var req struct {
		Name  string   `json:"name" binding:"required"`
		Icon  string   `json:"icon"`
		Color string   `json:"color"`
		Plaid []string `json:"plaid_primary_categories"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", err.Error()))
		return
	}
	if req.Plaid == nil {
		req.Plaid = []string{}
	}
	cat, err := h.repo.UpdateCategory(c.Param("id"), uid(c), req.Name, req.Icon, req.Color, req.Plaid)
	if err != nil {
		dbErr(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, cat)
}

// DELETE /v1/categories/:id
func (h *BudgetHandler) DeleteCategory(c *gin.Context) {
	if err := h.repo.DeleteCategory(c.Param("id"), uid(c)); err != nil {
		dbErr(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// PUT /v1/budget/:category_id?month=2026-06
func (h *BudgetHandler) SetBudget(c *gin.Context) {
	month, ok := parseMonth(c)
	if !ok {
		return
	}
	var req struct {
		Amount float64 `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", err.Error()))
		return
	}
	b, err := h.repo.SetBudget(uid(c), c.Param("category_id"), month, req.Amount)
	if err != nil {
		dbErr(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, b)
}
