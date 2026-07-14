package repositories

import (
	"database/sql"

	"github.com/vladyslavivchenko/netme/internal/crypto"
	"github.com/vladyslavivchenko/netme/internal/models"
)

type PlaidItemRepository struct {
	db  *sql.DB
	key []byte // AES-256 key; nil means encryption is disabled
}

func NewPlaidItemRepository(db *sql.DB, key []byte) *PlaidItemRepository {
	return &PlaidItemRepository{db: db, key: key}
}

func (r *PlaidItemRepository) encryptToken(plain string) (string, error) {
	if r.key == nil {
		return plain, nil
	}
	return crypto.Encrypt(r.key, plain)
}

func (r *PlaidItemRepository) decryptToken(stored string) (string, error) {
	if r.key == nil {
		return stored, nil
	}
	return crypto.Decrypt(r.key, stored)
}

func (r *PlaidItemRepository) CreateItem(userID, plaidItemID, accessToken string, institutionID, institutionName *string) (*models.PlaidItem, error) {
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

func (r *PlaidItemRepository) GetItemsByUserID(userID string) ([]*models.PlaidItem, error) {
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

func (r *PlaidItemRepository) GetItemByID(id string) (*models.PlaidItem, string, error) {
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
func (r *PlaidItemRepository) GetItemByPlaidItemID(plaidItemID string) (*models.PlaidItem, string, error) {
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

func (r *PlaidItemRepository) GetAllItemsForSync(userID string) ([]models.PlaidItemEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, plaid_item_id, access_token, institution_id, institution_name, cursor, created_at, updated_at
		 FROM plaid_items WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.PlaidItemEntry
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
		results = append(results, models.PlaidItemEntry{Item: item, AccessToken: accessToken})
	}
	return results, rows.Err()
}

func (r *PlaidItemRepository) UpdateCursor(itemID, cursor string) error {
	_, err := r.db.Exec(
		`UPDATE plaid_items SET cursor = $1, updated_at = now() WHERE id = $2`,
		cursor, itemID)
	return err
}

func (r *PlaidItemRepository) GetAllUserIDsWithItems() ([]string, error) {
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
