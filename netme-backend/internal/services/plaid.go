package services

import (
	"context"
	"fmt"

	plaid "github.com/plaid/plaid-go/v42/plaid"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type PlaidService struct {
	client    *plaid.APIClient
	plaidRepo *repositories.PlaidRepository
}

type SyncResult struct {
	Added      []plaid.Transaction
	Modified   []plaid.Transaction
	Removed    []plaid.RemovedTransaction
	NextCursor string
	HasMore    bool
}

func NewPlaidService(clientID, secret, env string, repo *repositories.PlaidRepository) *PlaidService {
	cfg := plaid.NewConfiguration()
	cfg.AddDefaultHeader("PLAID-CLIENT-ID", clientID)
	cfg.AddDefaultHeader("PLAID-SECRET", secret)

	if env == "production" {
		cfg.UseEnvironment(plaid.Production)
	} else {
		cfg.UseEnvironment(plaid.Sandbox)
	}

	return &PlaidService{client: plaid.NewAPIClient(cfg), plaidRepo: repo}
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

	item, err := s.plaidRepo.CreateItem(userID, itemID, accessToken, instID, instName)
	if err != nil {
		return nil, fmt.Errorf("store item: %w", err)
	}

	accounts, err := s.getAccounts(ctx, accessToken)
	if err == nil {
		s.plaidRepo.LogRawEvent(userID, "exchange_accounts", accounts)
		for _, pa := range accounts {
			_ = s.plaidRepo.UpsertAccount(accountFromPlaid(userID, item.ID, pa))
		}
	}

	s.plaidRepo.LogRawEvent(userID, "exchange", map[string]any{
		"item_id":          itemID,
		"institution_id":   institutionID,
		"institution_name": institutionName,
	})

	return item, nil
}

func (s *PlaidService) SyncForUser(ctx context.Context, userID string) (int, error) {
	items, err := s.plaidRepo.GetAllItemsForSync(userID)
	if err != nil {
		return 0, fmt.Errorf("load items: %w", err)
	}

	totalAdded := 0
	for _, entry := range items {
		cursor := ""
		if entry.Item.Cursor != nil {
			cursor = *entry.Item.Cursor
		}

		for {
			result, err := s.syncPage(ctx, entry.AccessToken, cursor)
			if err != nil {
				s.plaidRepo.LogRawEvent(userID, "sync_error", map[string]any{
					"error":   err.Error(),
					"item_id": entry.Item.PlaidItemID,
				})
				break
			}

			s.plaidRepo.LogRawEvent(userID, "sync_result", map[string]any{
				"item_id":     entry.Item.PlaidItemID,
				"added":       result.Added,
				"modified":    result.Modified,
				"removed":     result.Removed,
				"next_cursor": result.NextCursor,
				"has_more":    result.HasMore,
			})

			for _, pt := range result.Added {
				account, err := s.plaidRepo.GetAccountByPlaidID(pt.GetAccountId())
				if err != nil {
					continue
				}
				_ = s.plaidRepo.UpsertTransaction(txnFromPlaid(userID, account.ID, pt))
				totalAdded++
			}
			for _, pt := range result.Modified {
				account, err := s.plaidRepo.GetAccountByPlaidID(pt.GetAccountId())
				if err != nil {
					continue
				}
				_ = s.plaidRepo.UpsertTransaction(txnFromPlaid(userID, account.ID, pt))
			}
			for _, rt := range result.Removed {
				_ = s.plaidRepo.RemoveTransaction(rt.GetTransactionId())
			}

			cursor = result.NextCursor
			if !result.HasMore {
				break
			}
		}

		if cursor != "" {
			_ = s.plaidRepo.UpdateCursor(entry.Item.ID, cursor)
		}
	}

	return totalAdded, nil
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
