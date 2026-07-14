package repositories

import (
	"database/sql"
	"encoding/json"

	"github.com/vladyslavivchenko/netme/internal/crypto"
	"github.com/vladyslavivchenko/netme/internal/models"
)

type PlaidRepository struct {
	db  *sql.DB
	key []byte // AES-256 key; nil means encryption is disabled
}

func NewPlaidRepository(db *sql.DB, key []byte) *PlaidRepository {
	return &PlaidRepository{db: db, key: key}
}

func (r *PlaidRepository) encryptToken(plain string) (string, error) {
	if r.key == nil {
		return plain, nil
	}
	return crypto.Encrypt(r.key, plain)
}

func (r *PlaidRepository) decryptToken(stored string) (string, error) {
	if r.key == nil {
		return stored, nil
	}
	return crypto.Decrypt(r.key, stored)
}

func (r *PlaidRepository) CreateItem(userID, plaidItemID, accessToken string, institutionID, institutionName *string) (*models.PlaidItem, error) {
	storedToken, err := r.encryptToken(accessToken)
	if err != nil {
		return nil, err
	}
	item := &models.PlaidItem{}
	err = r.db.QueryRow(
		`INSERT INTO plaid_items (user_id, plaid_item_id, access_token, institution_id, institution_name)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, plaid_item_id, institution_id, institution_name, cursor, created_at, updated_at`,
		userID, plaidItemID, storedToken, institutionID, institutionName,
	).Scan(&item.ID, &item.UserID, &item.PlaidItemID, &item.InstitutionID, &item.InstitutionName, &item.Cursor, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (r *PlaidRepository) GetItemsByUserID(userID string) ([]*models.PlaidItem, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, plaid_item_id, institution_id, institution_name, cursor, created_at, updated_at
		 FROM plaid_items WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.PlaidItem
	for rows.Next() {
		item := &models.PlaidItem{}
		if err := rows.Scan(&item.ID, &item.UserID, &item.PlaidItemID, &item.InstitutionID, &item.InstitutionName, &item.Cursor, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PlaidRepository) GetItemByID(id string) (*models.PlaidItem, string, error) {
	item := &models.PlaidItem{}
	var storedToken string
	err := r.db.QueryRow(
		`SELECT id, user_id, plaid_item_id, access_token, institution_id, institution_name, cursor, created_at, updated_at
		 FROM plaid_items WHERE id = $1`, id,
	).Scan(&item.ID, &item.UserID, &item.PlaidItemID, &storedToken, &item.InstitutionID, &item.InstitutionName, &item.Cursor, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, "", err
	}
	accessToken, err := r.decryptToken(storedToken)
	return item, accessToken, err
}

// GetItemByPlaidItemID looks up an item by the external Plaid item_id (used by webhooks).
// Returns the item, its plaintext access token, and any error.
func (r *PlaidRepository) GetItemByPlaidItemID(plaidItemID string) (*models.PlaidItem, string, error) {
	item := &models.PlaidItem{}
	var storedToken string
	err := r.db.QueryRow(
		`SELECT id, user_id, plaid_item_id, access_token, institution_id, institution_name, cursor, created_at, updated_at
		 FROM plaid_items WHERE plaid_item_id = $1`, plaidItemID,
	).Scan(&item.ID, &item.UserID, &item.PlaidItemID, &storedToken,
		&item.InstitutionID, &item.InstitutionName, &item.Cursor, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, "", err
	}
	accessToken, err := r.decryptToken(storedToken)
	return item, accessToken, err
}

func (r *PlaidRepository) GetAllItemsForSync(userID string) ([]struct {
	Item        *models.PlaidItem
	AccessToken string
}, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, plaid_item_id, access_token, institution_id, institution_name, cursor, created_at, updated_at
		 FROM plaid_items WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		Item        *models.PlaidItem
		AccessToken string
	}
	for rows.Next() {
		item := &models.PlaidItem{}
		var storedToken string
		if err := rows.Scan(&item.ID, &item.UserID, &item.PlaidItemID, &storedToken, &item.InstitutionID, &item.InstitutionName, &item.Cursor, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		accessToken, err := r.decryptToken(storedToken)
		if err != nil {
			return nil, err
		}
		results = append(results, struct {
			Item        *models.PlaidItem
			AccessToken string
		}{item, accessToken})
	}
	return results, rows.Err()
}

func (r *PlaidRepository) UpdateCursor(itemID, cursor string) error {
	_, err := r.db.Exec(
		`UPDATE plaid_items SET cursor = $1, updated_at = now() WHERE id = $2`,
		cursor, itemID)
	return err
}

func (r *PlaidRepository) UpsertAccount(a *models.Account) error {
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

func (r *PlaidRepository) GetAccountsByUserID(userID string) ([]*models.Account, error) {
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

func (r *PlaidRepository) GetAccountByPlaidID(plaidAccountID, userID string) (*models.Account, error) {
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

func (r *PlaidRepository) UpsertTransaction(t *models.Transaction) error {
	_, err := r.db.Exec(
		`INSERT INTO transactions (user_id, account_id, plaid_transaction_id, amount, currency_code, name, merchant_name, date, authorized_date, category, category_detailed, payment_channel, pending)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 ON CONFLICT (plaid_transaction_id) DO UPDATE SET
		   amount = EXCLUDED.amount,
		   name = EXCLUDED.name,
		   merchant_name = EXCLUDED.merchant_name,
		   pending = EXCLUDED.pending,
		   updated_at = now()`,
		t.UserID, t.AccountID, t.PlaidTransactionID, t.Amount, t.CurrencyCode,
		t.Name, t.MerchantName, t.Date, t.AuthorizedDate,
		t.Category, t.CategoryDetailed, t.PaymentChannel, t.Pending,
	)
	return err
}

func (r *PlaidRepository) RemoveTransaction(plaidTransactionID string) error {
	_, err := r.db.Exec(`DELETE FROM transactions WHERE plaid_transaction_id = $1`, plaidTransactionID)
	return err
}

func (r *PlaidRepository) GetTransactionsByUserID(userID, accountID, month string, limit, offset int) ([]*models.Transaction, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, account_id, plaid_transaction_id, amount, currency_code, name, merchant_name,
		        to_char(date, 'YYYY-MM-DD'), to_char(authorized_date, 'YYYY-MM-DD'),
		        category, category_detailed, payment_channel, pending, category_id, created_at, updated_at
		 FROM transactions
		 WHERE user_id = $1
		   AND ($2 = '' OR account_id::text = $2)
		   AND ($3 = '' OR to_char(date, 'YYYY-MM') = $3)
		 ORDER BY date DESC, created_at DESC
		 LIMIT $4 OFFSET $5`, userID, accountID, month, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []*models.Transaction
	for rows.Next() {
		t := &models.Transaction{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.AccountID, &t.PlaidTransactionID, &t.Amount, &t.CurrencyCode,
			&t.Name, &t.MerchantName, &t.Date, &t.AuthorizedDate,
			&t.Category, &t.CategoryDetailed, &t.PaymentChannel, &t.Pending, &t.CategoryID,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

// GetAllUserIDsWithItems returns the distinct user IDs that have at least one Plaid item.
func (r *PlaidRepository) GetAllUserIDsWithItems() ([]string, error) {
	rows, err := r.db.Query(`SELECT DISTINCT user_id FROM plaid_items ORDER BY user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// TakeNetWorthSnapshot computes current net worth for a user and stores a daily snapshot.
// Uses ON CONFLICT to overwrite the same day's record if the job runs more than once.
func (r *PlaidRepository) TakeNetWorthSnapshot(userID string) error {
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

func (r *PlaidRepository) GetNetWorth(userID string) (*models.NetWorth, error) {
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

// PurgeOldRawEvents deletes raw event rows older than the given number of days.
// Call weekly to cap table growth (GDPR Article 5(1)(e)).
func (r *PlaidRepository) PurgeOldRawEvents(olderThanDays int) (int64, error) {
	res, err := r.db.Exec(
		`DELETE FROM plaid_raw_events WHERE created_at < now() - ($1::int * interval '1 day')`,
		olderThanDays,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// LogRawEvent stores any Plaid payload for debugging. userID may be empty for webhook events.
func (r *PlaidRepository) LogRawEvent(userID, eventType string, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	var uid *string
	if userID != "" {
		uid = &userID
	}
	_, _ = r.db.Exec(
		`INSERT INTO plaid_raw_events (user_id, event_type, payload) VALUES ($1, $2, $3)`,
		uid, eventType, string(b),
	)
}

func (r *PlaidRepository) GetTransactionByID(userID, id string) (*models.Transaction, error) {
	t := &models.Transaction{}
	err := r.db.QueryRow(
		`SELECT id, user_id, account_id, plaid_transaction_id, amount, currency_code, name, merchant_name,
		        to_char(date, 'YYYY-MM-DD'), to_char(authorized_date, 'YYYY-MM-DD'),
		        category, category_detailed, payment_channel, pending, category_id, created_at, updated_at
		 FROM transactions WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&t.ID, &t.UserID, &t.AccountID, &t.PlaidTransactionID, &t.Amount, &t.CurrencyCode,
		&t.Name, &t.MerchantName, &t.Date, &t.AuthorizedDate,
		&t.Category, &t.CategoryDetailed, &t.PaymentChannel, &t.Pending, &t.CategoryID,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *PlaidRepository) PatchTransactionCategory(userID, txnID, categoryID string) (*models.Transaction, error) {
	t := &models.Transaction{}
	err := r.db.QueryRow(
		`UPDATE transactions SET category_id = $1, updated_at = now()
		 WHERE id = $2 AND user_id = $3
		   AND EXISTS (SELECT 1 FROM categories WHERE id = $1 AND user_id = $3)
		 RETURNING id, user_id, account_id, plaid_transaction_id, amount, currency_code, name, merchant_name,
		           to_char(date, 'YYYY-MM-DD'), to_char(authorized_date, 'YYYY-MM-DD'),
		           category, category_detailed, payment_channel, pending, category_id, created_at, updated_at`,
		categoryID, txnID, userID,
	).Scan(&t.ID, &t.UserID, &t.AccountID, &t.PlaidTransactionID, &t.Amount, &t.CurrencyCode,
		&t.Name, &t.MerchantName, &t.Date, &t.AuthorizedDate,
		&t.Category, &t.CategoryDetailed, &t.PaymentChannel, &t.Pending, &t.CategoryID,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}
