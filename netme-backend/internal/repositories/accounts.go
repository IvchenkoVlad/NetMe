package repositories

import (
	"database/sql"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) UpsertAccount(a *models.Account) error {
	_, err := r.db.Exec(
		`INSERT INTO accounts (user_id, plaid_item_id, plaid_account_id, name, official_name, type, subtype, mask, current_balance, available_balance, currency_code)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (plaid_account_id) DO UPDATE SET
		   name = EXCLUDED.name,
		   current_balance = EXCLUDED.current_balance,
		   available_balance = EXCLUDED.available_balance,
		   updated_at = now()`,
		a.UserID, a.PlaidItemID, a.PlaidAccountID, a.Name, a.OfficialName,
		a.Type, a.Subtype, a.Mask, a.CurrentBalance, a.AvailableBalance, a.CurrencyCode,
	)
	return err
}

func (r *AccountRepository) GetAccountsByUserID(userID string) ([]*models.Account, error) {
	rows, err := r.db.Query(
		`SELECT a.id, a.user_id, a.plaid_item_id, a.plaid_account_id, a.name, a.official_name,
		        a.type, a.subtype, a.mask, a.current_balance, a.available_balance, a.currency_code,
		        a.created_at, a.updated_at,
		        COALESCE(pi.institution_name, 'Unknown Bank') AS institution_name
		 FROM accounts a
		 LEFT JOIN plaid_items pi ON pi.id = a.plaid_item_id
		 WHERE a.user_id = $1
		 ORDER BY COALESCE(pi.institution_name, ''), a.type, a.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.Account
	for rows.Next() {
		a := &models.Account{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.PlaidItemID, &a.PlaidAccountID, &a.Name, &a.OfficialName,
			&a.Type, &a.Subtype, &a.Mask, &a.CurrentBalance, &a.AvailableBalance, &a.CurrencyCode,
			&a.CreatedAt, &a.UpdatedAt, &a.InstitutionName); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (r *AccountRepository) GetAccountByPlaidID(plaidAccountID, userID string) (*models.Account, error) {
	a := &models.Account{}
	err := r.db.QueryRow(
		`SELECT id, user_id, plaid_item_id, plaid_account_id, name, official_name, type, subtype, mask,
		        current_balance, available_balance, currency_code, created_at, updated_at
		 FROM accounts WHERE plaid_account_id = $1 AND user_id = $2`, plaidAccountID, userID,
	).Scan(&a.ID, &a.UserID, &a.PlaidItemID, &a.PlaidAccountID, &a.Name, &a.OfficialName,
		&a.Type, &a.Subtype, &a.Mask, &a.CurrentBalance, &a.AvailableBalance, &a.CurrencyCode,
		&a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (r *AccountRepository) GetNetWorth(userID string) (*models.NetWorth, error) {
	var assets, liabilities float64
	err := r.db.QueryRow(
		`SELECT
		   COALESCE(SUM(CASE WHEN type IN ('depository','investment') THEN COALESCE(current_balance,0) ELSE 0 END), 0),
		   COALESCE(SUM(CASE WHEN type IN ('credit','loan')           THEN COALESCE(current_balance,0) ELSE 0 END), 0)
		 FROM accounts WHERE user_id = $1`, userID,
	).Scan(&assets, &liabilities)
	if err != nil {
		return nil, err
	}
	return &models.NetWorth{
		Assets:      assets,
		Liabilities: liabilities,
		NetWorth:    assets - liabilities,
	}, nil
}

func (r *AccountRepository) TakeNetWorthSnapshot(userID string) error {
	nw, err := r.GetNetWorth(userID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		`INSERT INTO net_worth_snapshots (user_id, assets, liabilities, net_worth, recorded_at)
		 VALUES ($1, $2, $3, $4, CURRENT_DATE)
		 ON CONFLICT (user_id, recorded_at) DO UPDATE
		   SET assets=EXCLUDED.assets, liabilities=EXCLUDED.liabilities, net_worth=EXCLUDED.net_worth`,
		userID, nw.Assets, nw.Liabilities, nw.NetWorth,
	)
	return err
}
