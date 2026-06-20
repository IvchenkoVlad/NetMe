package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	plaidgo "github.com/plaid/plaid-go/v42/plaid"
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
	// link-page is public — the link_token itself is the credential
	public.GET("/plaid/link-page", h.LinkPage)
}

func (h *PlaidHandler) CreateLinkToken(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	token, err := h.plaidSvc.CreateLinkToken(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "plaid_error",
			Message: "failed to create link token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"link_token": token})
}

func (h *PlaidHandler) ExchangeToken(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	var req struct {
		PublicToken     string `json:"public_token" binding:"required"`
		InstitutionID   string `json:"institution_id"`
		InstitutionName string `json:"institution_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	accessToken, itemID, err := h.plaidSvc.ExchangePublicToken(ctx, req.PublicToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "plaid_error",
			Message: "failed to exchange token",
		})
		return
	}

	var instID, instName *string
	if req.InstitutionID != "" {
		instID = &req.InstitutionID
	}
	if req.InstitutionName != "" {
		instName = &req.InstitutionName
	}

	item, err := h.plaidRepo.CreateItem(uid, itemID, accessToken, instID, instName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "failed to store item",
		})
		return
	}

	// Immediately fetch and store accounts
	plaidAccounts, err := h.plaidSvc.GetAccounts(ctx, accessToken)
	if err == nil {
		h.plaidRepo.LogRawEvent(uid, "exchange_accounts", plaidAccounts)
		for _, pa := range plaidAccounts {
			a := accountFromPlaid(uid, item.ID, pa)
			_ = h.plaidRepo.UpsertAccount(a)
		}
	}

	h.plaidRepo.LogRawEvent(uid, "exchange", map[string]any{
		"item_id":          itemID,
		"institution_id":   req.InstitutionID,
		"institution_name": req.InstitutionName,
	})

	c.JSON(http.StatusOK, gin.H{"item": item})
}

func (h *PlaidHandler) SyncTransactions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	ctx := c.Request.Context()

	items, err := h.plaidRepo.GetAllItemsForSync(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "failed to load items",
		})
		return
	}

	totalAdded := 0
	for _, entry := range items {
		cursor := ""
		if entry.Item.Cursor != nil {
			cursor = *entry.Item.Cursor
		}

		for {
			result, err := h.plaidSvc.SyncTransactions(ctx, entry.AccessToken, cursor)
			if err != nil {
				h.plaidRepo.LogRawEvent(uid, "sync_error", map[string]any{"error": err.Error(), "item_id": entry.Item.PlaidItemID})
				break
			}

			h.plaidRepo.LogRawEvent(uid, "sync_result", map[string]any{
				"item_id":     entry.Item.PlaidItemID,
				"added":       result.Added,
				"modified":    result.Modified,
				"removed":     result.Removed,
				"next_cursor": result.NextCursor,
				"has_more":    result.HasMore,
			})

			for _, pt := range result.Added {
				account, err := h.plaidRepo.GetAccountByPlaidID(pt.GetAccountId())
				if err != nil {
					continue
				}
				_ = h.plaidRepo.UpsertTransaction(txnFromPlaid(uid, account.ID, pt))
				totalAdded++
			}
			for _, pt := range result.Modified {
				account, err := h.plaidRepo.GetAccountByPlaidID(pt.GetAccountId())
				if err != nil {
					continue
				}
				_ = h.plaidRepo.UpsertTransaction(txnFromPlaid(uid, account.ID, pt))
			}
			for _, rt := range result.Removed {
				_ = h.plaidRepo.RemoveTransaction(rt.GetTransactionId())
			}

			cursor = result.NextCursor
			if !result.HasMore {
				break
			}
		}

		if cursor != "" {
			_ = h.plaidRepo.UpdateCursor(entry.Item.ID, cursor)
		}
	}

	c.JSON(http.StatusOK, gin.H{"transactions_added": totalAdded})
}

func (h *PlaidHandler) ListItems(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	items, err := h.plaidRepo.GetItemsByUserID(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "failed to load items",
		})
		return
	}
	if items == nil {
		items = []*models.PlaidItem{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
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

func accountFromPlaid(userID, plaidItemID string, pa plaidgo.AccountBase) *models.Account {
	a := &models.Account{
		UserID:         userID,
		PlaidItemID:    plaidItemID,
		PlaidAccountID: pa.GetAccountId(),
		Name:           pa.GetName(),
		Type:           string(pa.GetType()),
		CurrencyCode:   "USD",
	}
	if name := pa.GetOfficialName(); name != "" {
		a.OfficialName = &name
	}
	if sub := string(pa.GetSubtype()); sub != "" {
		a.Subtype = &sub
	}
	if mask := pa.GetMask(); mask != "" {
		a.Mask = &mask
	}
	balances := pa.GetBalances()
	if cur := balances.GetCurrent(); cur != 0 {
		a.CurrentBalance = &cur
	}
	if avail := balances.GetAvailable(); avail != 0 {
		a.AvailableBalance = &avail
	}
	if code := balances.GetIsoCurrencyCode(); code != "" {
		a.CurrencyCode = code
	}
	return a
}

func txnFromPlaid(userID, accountID string, pt plaidgo.Transaction) *models.Transaction {
	t := &models.Transaction{
		UserID:             userID,
		AccountID:          accountID,
		PlaidTransactionID: pt.GetTransactionId(),
		Amount:             pt.GetAmount(),
		CurrencyCode:       "USD",
		Name:               pt.GetName(),
		Date:               pt.GetDate(),
		Pending:            pt.GetPending(),
	}
	if code := pt.GetIsoCurrencyCode(); code != "" {
		t.CurrencyCode = code
	}
	if m := pt.GetMerchantName(); m != "" {
		t.MerchantName = &m
	}
	if d := string(pt.GetAuthorizedDate()); d != "" && d != "0001-01-01" {
		t.AuthorizedDate = &d
	}
	if ch := string(pt.GetPaymentChannel()); ch != "" {
		t.PaymentChannel = &ch
	}
	if cats := pt.GetPersonalFinanceCategory(); true {
		if pri := cats.GetPrimary(); pri != "" {
			t.Category = &pri
		}
		if det := cats.GetDetailed(); det != "" {
			t.CategoryDetailed = &det
		}
	}
	return t
}
