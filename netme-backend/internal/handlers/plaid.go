package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

type PlaidHandler struct {
	plaidSvc  *services.PlaidService
	plaidRepo *repositories.PlaidRepository
}

func NewPlaidHandler(svc *services.PlaidService, repo *repositories.PlaidRepository) *PlaidHandler {
	return &PlaidHandler{plaidSvc: svc, plaidRepo: repo}
}

func RegisterPlaidRoutes(r *gin.RouterGroup, public *gin.RouterGroup, svc *services.PlaidService, repo *repositories.PlaidRepository) {
	h := NewPlaidHandler(svc, repo)
	plaid := r.Group("/plaid")
	{
		plaid.POST("/link-token", h.CreateLinkToken)
		plaid.POST("/exchange", h.ExchangeToken)
		plaid.POST("/sync", h.SyncTransactions)
		plaid.GET("/items", h.ListItems)
	}
	public.GET("/plaid/link-page", h.LinkPage)
	// Webhook is unauthenticated — Plaid calls it directly.
	// TODO: add Plaid JWT signature verification before production launch.
	public.POST("/plaid/webhook", h.Webhook)
}

func (h *PlaidHandler) CreateLinkToken(c *gin.Context) {
	token, err := h.plaidSvc.CreateLinkToken(c.Request.Context(), uid(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResp("plaid_error", "failed to create link token"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"link_token": token})
}

func (h *PlaidHandler) ExchangeToken(c *gin.Context) {
	var req struct {
		PublicToken     string `json:"public_token" binding:"required"`
		InstitutionID   string `json:"institution_id"`
		InstitutionName string `json:"institution_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp("invalid_request", err.Error()))
		return
	}

	item, err := h.plaidSvc.ExchangeAndStore(c.Request.Context(), uid(c), req.PublicToken, req.InstitutionID, req.InstitutionName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResp("plaid_error", "failed to exchange token"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item})
}

func (h *PlaidHandler) SyncTransactions(c *gin.Context) {
	totalAdded, err := h.plaidSvc.SyncForUser(c.Request.Context(), uid(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResp("database_error", "failed to load items"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"transactions_added": totalAdded})
}

func (h *PlaidHandler) ListItems(c *gin.Context) {
	items, err := h.plaidRepo.GetItemsByUserID(uid(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResp("database_error", "failed to load items"))
		return
	}
	if items == nil {
		items = []*models.PlaidItem{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Webhook handles incoming Plaid webhook events.
// It responds immediately with 200 and processes the event asynchronously.
//
// Supported event types:
//   - TRANSACTIONS / SYNC_UPDATES_AVAILABLE, DEFAULT_UPDATE, INITIAL_UPDATE → trigger item sync
//   - ITEM / ERROR → log for investigation
func (h *PlaidHandler) Webhook(c *gin.Context) {
	// Read raw body so we can verify the signature before parsing.
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if err := h.plaidSvc.WebhookVerifier.Verify(
		c.GetHeader("Plaid-Verification"), body,
	); err != nil {
		c.JSON(http.StatusUnauthorized, errResp("invalid_signature", err.Error()))
		return
	}

	var payload struct {
		WebhookType string `json:"webhook_type"`
		WebhookCode string `json:"webhook_code"`
		ItemID      string `json:"item_id"`
		Error       *struct {
			ErrorType    string `json:"error_type"`
			ErrorCode    string `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	h.plaidRepo.LogRawEvent("", "webhook_received", map[string]any{
		"type": payload.WebhookType,
		"code": payload.WebhookCode,
		"item": payload.ItemID,
	})

	// Respond immediately so Plaid doesn't retry.
	c.Status(http.StatusOK)

	go func() {
		ctx := context.Background()
		switch payload.WebhookType {
		case "TRANSACTIONS":
			switch payload.WebhookCode {
			case "SYNC_UPDATES_AVAILABLE", "DEFAULT_UPDATE", "INITIAL_UPDATE", "RECURRING_TRANSACTIONS_UPDATE":
				item, _, err := h.plaidRepo.GetItemByPlaidItemID(payload.ItemID)
				if err != nil {
					return
				}
				_, _ = h.plaidSvc.SyncItem(ctx, item.UserID, item.ID)
			}
		case "ITEM":
			if payload.Error != nil {
				h.plaidRepo.LogRawEvent("", "webhook_item_error", map[string]any{
					"item_id":       payload.ItemID,
					"error_type":    payload.Error.ErrorType,
					"error_code":    payload.Error.ErrorCode,
					"error_message": payload.Error.ErrorMessage,
				})
			}
		}
	}()
}

func (h *PlaidHandler) LinkPage(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "missing token")
		return
	}
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
  <title>NetMe - Connect Bank</title>
  <style>
    body { margin: 0; background: #0f172a; display: flex; align-items: center; justify-content: center; height: 100vh; }
    p { color: #2dd4a7; font-family: sans-serif; font-size: 16px; }
  </style>
</head>
<body>
  <p>Opening Plaid Link…</p>
  <script src="https://cdn.plaid.com/link/v2/stable/link-initialize.js"></script>
  <script>
  (function() {
    function send(obj) {
      var msg = JSON.stringify(obj);
      if (window.ReactNativeWebView) {
        window.ReactNativeWebView.postMessage(msg);
      } else {
        window.location.href = 'plaidlink://callback?data=' + encodeURIComponent(msg);
      }
    }

    var handler = Plaid.create({
      token: '%s',
      onSuccess: function(public_token, metadata) {
        send({
          event: 'success',
          public_token: public_token,
          institution_id: (metadata.institution || {}).institution_id || '',
          institution_name: (metadata.institution || {}).name || ''
        });
      },
      onExit: function(err, metadata) {
        send({ event: 'exit', error: err ? err.error_code : null });
      },
      onEvent: function(eventName, metadata) {}
    });

    handler.open();
  })();
  </script>
</body>
</html>`, token)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
