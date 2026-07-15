package services

import (
	"context"
	"fmt"

	plaid "github.com/plaid/plaid-go/v42/plaid"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type PlaidService struct {
	client          *plaid.APIClient
	itemRepo        *repositories.PlaidItemRepository
	acctRepo        *repositories.AccountRepository
	txnRepo         *repositories.TransactionRepository
	eventRepo       *repositories.EventRepository
	rulesRepo       *repositories.RulesRepository
	WebhookVerifier *WebhookVerifier
}

type SyncResult struct {
	Added      []plaid.Transaction
	Modified   []plaid.Transaction
	Removed    []plaid.RemovedTransaction
	NextCursor string
	HasMore    bool
}

func NewPlaidService(
	clientID, secret, env string,
	itemRepo *repositories.PlaidItemRepository,
	acctRepo *repositories.AccountRepository,
	txnRepo *repositories.TransactionRepository,
	eventRepo *repositories.EventRepository,
	rulesRepo *repositories.RulesRepository,
) *PlaidService {
	cfg := plaid.NewConfiguration()
	cfg.AddDefaultHeader("PLAID-CLIENT-ID", clientID)
	cfg.AddDefaultHeader("PLAID-SECRET", secret)

	if env == "production" {
		cfg.UseEnvironment(plaid.Production)
	} else {
		cfg.UseEnvironment(plaid.Sandbox)
	}

	client := plaid.NewAPIClient(cfg)
	return &PlaidService{
		client:          client,
		itemRepo:        itemRepo,
		acctRepo:        acctRepo,
		txnRepo:         txnRepo,
		eventRepo:       eventRepo,
		rulesRepo:       rulesRepo,
		WebhookVerifier: NewWebhookVerifier(client, env),
	}
}

func (s *PlaidService) CreateLinkToken(ctx context.Context, userID string) (string, error) {
	user := plaid.NewLinkTokenCreateRequestUser(userID)
	req := plaid.NewLinkTokenCreateRequest("NetMe", "en", []plaid.CountryCode{plaid.COUNTRYCODE_US})
	req.SetUser(*user)
	req.SetProducts([]plaid.Products{plaid.PRODUCTS_TRANSACTIONS})

	resp, _, err := s.client.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*req).Execute()
	if err != nil {
		return "", fmt.Errorf("link token create: %w", err)
	}
	return resp.GetLinkToken(), nil
}

func (s *PlaidService) ExchangeAndStore(ctx context.Context, userID, publicToken, institutionID, institutionName string) (*models.PlaidItem, error) {
	req := plaid.NewItemPublicTokenExchangeRequest(publicToken)
	resp, _, err := s.client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(*req).Execute()
	if err != nil {
		return nil, fmt.Errorf("exchange public token: %w", err)
	}
	accessToken, itemID := resp.GetAccessToken(), resp.GetItemId()

	var instID, instName *string
	if institutionID != "" {
		instID = &institutionID
	}
	if institutionName != "" {
		instName = &institutionName
	}

	item, err := s.itemRepo.CreateItem(userID, itemID, accessToken, instID, instName)
	if err != nil {
		return nil, fmt.Errorf("store item: %w", err)
	}

	accounts, err := s.getAccounts(ctx, accessToken)
	if err == nil {
		s.eventRepo.LogRawEvent(userID, "exchange_accounts", accounts)
		for _, pa := range accounts {
			_ = s.acctRepo.UpsertAccount(accountFromPlaid(userID, item.ID, pa))
		}
	}

	s.eventRepo.LogRawEvent(userID, "exchange", map[string]any{
		"item_id":          itemID,
		"institution_id":   institutionID,
		"institution_name": institutionName,
	})

	return item, nil
}

// SyncItem syncs a single Plaid item identified by its internal UUID.
// Used by the webhook handler so only the affected item is synced.
func (s *PlaidService) SyncItem(ctx context.Context, userID, itemUUID string) (int, error) {
	item, accessToken, err := s.itemRepo.GetItemByID(itemUUID)
	if err != nil {
		return 0, fmt.Errorf("load item: %w", err)
	}
	added, finalCursor, err := s.drainPages(ctx, userID, item.PlaidItemID, accessToken, derefCursor(item.Cursor))
	if err != nil {
		return added, err
	}
	if finalCursor != "" {
		_ = s.itemRepo.UpdateCursor(itemUUID, finalCursor)
	}
	if s.rulesRepo != nil {
		_ = s.rulesRepo.ApplyCategoryRules(userID)
	}
	return added, nil
}

// SyncForUser syncs all Plaid items for a user in sequence.
func (s *PlaidService) SyncForUser(ctx context.Context, userID string) (int, error) {
	items, err := s.itemRepo.GetAllItemsForSync(userID)
	if err != nil {
		return 0, fmt.Errorf("load items: %w", err)
	}
	totalAdded := 0
	for _, entry := range items {
		added, finalCursor, err := s.drainPages(ctx, userID, entry.Item.PlaidItemID, entry.AccessToken, derefCursor(entry.Item.Cursor))
		totalAdded += added
		if err != nil {
			continue // already logged inside drainPages
		}
		if finalCursor != "" {
			_ = s.itemRepo.UpdateCursor(entry.Item.ID, finalCursor)
		}
	}
	if s.rulesRepo != nil {
		_ = s.rulesRepo.ApplyCategoryRules(userID)
	}
	return totalAdded, nil
}

// drainPages runs the cursor-paginated sync loop for one Plaid item until HasMore is false.
// Returns (transactions added, final cursor, first error encountered).
func (s *PlaidService) drainPages(ctx context.Context, userID, plaidItemID, accessToken, cursor string) (int, string, error) {
	added := 0
	for {
		result, err := s.syncPage(ctx, accessToken, cursor)
		if err != nil {
			s.eventRepo.LogRawEvent(userID, "sync_error", map[string]any{
				"item_id": plaidItemID,
				"error":   err.Error(),
			})
			return added, cursor, err
		}
		s.eventRepo.LogRawEvent(userID, "sync_result", map[string]any{
			"item_id":  plaidItemID,
			"added":    len(result.Added),
			"modified": len(result.Modified),
			"removed":  len(result.Removed),
		})
		for _, pt := range result.Added {
			acct, err := s.acctRepo.GetAccountByPlaidID(pt.GetAccountId(), userID)
			if err != nil {
				continue
			}
			_ = s.txnRepo.UpsertTransaction(txnFromPlaid(userID, acct.ID, pt))
			added++
		}
		for _, pt := range result.Modified {
			acct, err := s.acctRepo.GetAccountByPlaidID(pt.GetAccountId(), userID)
			if err != nil {
				continue
			}
			_ = s.txnRepo.UpsertTransaction(txnFromPlaid(userID, acct.ID, pt))
		}
		for _, rt := range result.Removed {
			_ = s.txnRepo.RemoveTransaction(rt.GetTransactionId())
		}
		cursor = result.NextCursor
		if !result.HasMore {
			break
		}
	}
	return added, cursor, nil
}

// RevokeAllItems calls Plaid's /item/remove for every item owned by the user.
// Errors are logged but do not block account deletion — the DB cascade handles cleanup.
func (s *PlaidService) RevokeAllItems(ctx context.Context, userID string) {
	items, err := s.itemRepo.GetAllItemsForSync(userID)
	if err != nil {
		return
	}
	for _, entry := range items {
		req := plaid.NewItemRemoveRequest(entry.AccessToken)
		_, _, _ = s.client.PlaidApi.ItemRemove(ctx).ItemRemoveRequest(*req).Execute()
	}
}

func (s *PlaidService) GetInstitution(ctx context.Context, institutionID string) (string, error) {
	req := plaid.NewInstitutionsGetByIdRequest(institutionID, []plaid.CountryCode{plaid.COUNTRYCODE_US})
	resp, _, err := s.client.PlaidApi.InstitutionsGetById(ctx).InstitutionsGetByIdRequest(*req).Execute()
	if err != nil {
		return "", nil // non-fatal
	}
	return resp.Institution.GetName(), nil
}

func (s *PlaidService) syncPage(ctx context.Context, accessToken, cursor string) (*SyncResult, error) {
	req := plaid.NewTransactionsSyncRequest(accessToken)
	if cursor != "" {
		req.SetCursor(cursor)
	}
	resp, _, err := s.client.PlaidApi.TransactionsSync(ctx).TransactionsSyncRequest(*req).Execute()
	if err != nil {
		return nil, fmt.Errorf("transactions sync: %w", err)
	}
	return &SyncResult{
		Added:      resp.GetAdded(),
		Modified:   resp.GetModified(),
		Removed:    resp.GetRemoved(),
		NextCursor: resp.GetNextCursor(),
		HasMore:    resp.GetHasMore(),
	}, nil
}

func (s *PlaidService) getAccounts(ctx context.Context, accessToken string) ([]plaid.AccountBase, error) {
	req := plaid.NewAccountsGetRequest(accessToken)
	resp, _, err := s.client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*req).Execute()
	if err != nil {
		return nil, fmt.Errorf("accounts get: %w", err)
	}
	return resp.GetAccounts(), nil
}

func derefCursor(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func accountFromPlaid(userID, plaidItemID string, pa plaid.AccountBase) *models.Account {
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

func txnFromPlaid(userID, accountID string, pt plaid.Transaction) *models.Transaction {
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
	cats := pt.GetPersonalFinanceCategory()
	if pri := cats.GetPrimary(); pri != "" {
		t.Category = &pri
	}
	if det := cats.GetDetailed(); det != "" {
		t.CategoryDetailed = &det
	}
	return t
}
