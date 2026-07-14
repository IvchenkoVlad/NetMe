package repositories

import (
	"database/sql"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// scanner is satisfied by both *sql.Row and *sql.Rows, allowing one
// scan implementation to serve both single-row and multi-row queries.
type scanner interface {
	Scan(dest ...any) error
}

// scanTransaction reads the standard 17-column transaction projection.
// Column order must match every SELECT that uses this helper.
func scanTransaction(s scanner) (*models.Transaction, error) {
	t := &models.Transaction{}
	err := s.Scan(
		&t.ID, &t.UserID, &t.AccountID, &t.PlaidTransactionID,
		&t.Amount, &t.CurrencyCode, &t.Name, &t.MerchantName,
		&t.Date, &t.AuthorizedDate,
		&t.Category, &t.CategoryDetailed, &t.PaymentChannel,
		&t.Pending, &t.CategoryID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}

const txnColumns = `id, user_id, account_id, plaid_transaction_id,
	amount, currency_code, name, merchant_name,
	to_char(date, 'YYYY-MM-DD'), to_char(authorized_date, 'YYYY-MM-DD'),
	category, category_detailed, payment_channel,
	pending, category_id, created_at, updated_at`

func (r *TransactionRepository) UpsertTransaction(t *models.Transaction) error {
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

func (r *TransactionRepository) RemoveTransaction(plaidTransactionID string) error {
	_, err := r.db.Exec(`DELETE FROM transactions WHERE plaid_transaction_id = $1`, plaidTransactionID)
	return err
}

func (r *TransactionRepository) GetTransactionsByUserID(userID, accountID, month string, limit, offset int) ([]*models.Transaction, error) {
	rows, err := r.db.Query(
		`SELECT `+txnColumns+`
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
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

func (r *TransactionRepository) GetTransactionByID(userID, id string) (*models.Transaction, error) {
	row := r.db.QueryRow(
		`SELECT `+txnColumns+`
		 FROM transactions WHERE id = $1 AND user_id = $2`, id, userID)
	t, err := scanTransaction(row)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TransactionRepository) PatchTransactionCategory(userID, txnID, categoryID string) (*models.Transaction, error) {
	row := r.db.QueryRow(
		`UPDATE transactions SET category_id = $1, updated_at = now()
		 WHERE id = $2 AND user_id = $3
		   AND EXISTS (SELECT 1 FROM categories WHERE id = $1 AND user_id = $3)
		 RETURNING `+txnColumns,
		categoryID, txnID, userID)
	t, err := scanTransaction(row)
	if err != nil {
		return nil, err
	}
	return t, nil
}
