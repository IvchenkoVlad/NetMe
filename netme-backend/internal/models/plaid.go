package models

import "time"

type PlaidItem struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	PlaidItemID     string    `json:"plaid_item_id"`
	InstitutionID   *string   `json:"institution_id,omitempty"`
	InstitutionName *string   `json:"institution_name,omitempty"`
	Cursor          *string   `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// PlaidItemEntry pairs a PlaidItem with its decrypted access token.
// Used as the return type of GetAllItemsForSync.
type PlaidItemEntry struct {
	Item        *PlaidItem
	AccessToken string
}
