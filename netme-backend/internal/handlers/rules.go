package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
)

// RulesRepo is the subset of RulesRepository used by rules handlers.
type RulesRepo interface {
	Upsert(userID, normalizedMerchant, categoryID string) (*models.CategoryRule, error)
	ApplyToPast(userID, normalizedMerchant, categoryID string) (int64, error)
	List(userID string) ([]*models.CategoryRule, error)
	Delete(userID, id string) error
}

type RulesHandler struct {
	repo RulesRepo
}

func NewRulesHandler(repo RulesRepo) *RulesHandler {
	return &RulesHandler{repo: repo}
}

func RegisterRulesRoutes(r *gin.RouterGroup, repo RulesRepo) {
	h := NewRulesHandler(repo)
	rules := r.Group("/rules")
	{
		rules.POST("", h.CreateRule)
		rules.GET("", h.ListRules)
		rules.DELETE("/:id", h.DeleteRule)
	}
}

func (h *RulesHandler) CreateRule(c *gin.Context) {
	var req struct {
		NormalizedMerchant string `json:"normalized_merchant" binding:"required"`
		CategoryID         string `json:"category_id" binding:"required"`
		ApplyToPast        bool   `json:"apply_to_past"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", err.Error()))
		return
	}

	userID := uid(c)
	rule, err := h.repo.Upsert(userID, req.NormalizedMerchant, req.CategoryID)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, errResp("not_found", "category not found"))
		return
	}
	if err != nil {
		dbErr(c, "failed to save rule")
		return
	}

	var updatedCount int64
	if req.ApplyToPast {
		updatedCount, err = h.repo.ApplyToPast(userID, req.NormalizedMerchant, req.CategoryID)
		if err != nil {
			dbErr(c, "failed to apply rule to past")
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"rule": rule, "updated_count": updatedCount})
}

func (h *RulesHandler) ListRules(c *gin.Context) {
	rules, err := h.repo.List(uid(c))
	if err != nil {
		dbErr(c, "failed to load rules")
		return
	}
	if rules == nil {
		rules = []*models.CategoryRule{}
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (h *RulesHandler) DeleteRule(c *gin.Context) {
	err := h.repo.Delete(uid(c), c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, errResp("not_found", "rule not found"))
		return
	}
	if err != nil {
		dbErr(c, "failed to delete rule")
		return
	}
	c.Status(http.StatusNoContent)
}
