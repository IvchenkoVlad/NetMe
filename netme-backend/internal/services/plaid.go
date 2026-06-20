package services

import (
	"context"
	"fmt"

	plaid "github.com/plaid/plaid-go/v42/plaid"
)

type PlaidService struct {
	client *plaid.APIClient
}

type SyncResult struct {
	Added       []plaid.Transaction
	Modified    []plaid.Transaction
	Removed     []plaid.RemovedTransaction
	NextCursor  string
	HasMore     bool
}

func NewPlaidService(clientID, secret, env string) *PlaidService {
	cfg := plaid.NewConfiguration()
	cfg.AddDefaultHeader("PLAID-CLIENT-ID", clientID)
	cfg.AddDefaultHeader("PLAID-SECRET", secret)

	if env == "production" {
		cfg.UseEnvironment(plaid.Production)
	} else {
		cfg.UseEnvironment(plaid.Sandbox)
	}

	return &PlaidService{client: plaid.NewAPIClient(cfg)}
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

func (s *PlaidService) ExchangePublicToken(ctx context.Context, publicToken string) (accessToken, itemID string, err error) {
	req := plaid.NewItemPublicTokenExchangeRequest(publicToken)
	resp, _, err := s.client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(*req).Execute()
	if err != nil {
		return "", "", fmt.Errorf("exchange public token: %w", err)
	}
	return resp.GetAccessToken(), resp.GetItemId(), nil
}

func (s *PlaidService) GetAccounts(ctx context.Context, accessToken string) ([]plaid.AccountBase, error) {
	req := plaid.NewAccountsGetRequest(accessToken)
	resp, _, err := s.client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*req).Execute()
	if err != nil {
		return nil, fmt.Errorf("accounts get: %w", err)
	}
	return resp.GetAccounts(), nil
}

func (s *PlaidService) GetInstitution(ctx context.Context, institutionID string) (string, error) {
	req := plaid.NewInstitutionsGetByIdRequest(institutionID, []plaid.CountryCode{plaid.COUNTRYCODE_US})
	resp, _, err := s.client.PlaidApi.InstitutionsGetById(ctx).InstitutionsGetByIdRequest(*req).Execute()
	if err != nil {
		return "", nil // non-fatal
	}
	return resp.Institution.GetName(), nil
}

func (s *PlaidService) SyncTransactions(ctx context.Context, accessToken, cursor string) (*SyncResult, error) {
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
