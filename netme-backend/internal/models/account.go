package models

import "time"

type Account struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	PlaidItemID      string    `json:"plaid_item_id"`
	PlaidAccountID   string    `json:"plaid_account_id"`
	Name             string    `json:"name"`
	OfficialName     *string   `json:"official_name,omitempty"`
	Type             string    `json:"type"`
	Subtype          *string   `json:"subtype,omitempty"`
	Mask             *string   `json:"mask,omitempty"`
	CurrentBalance   *float64  `json:"current_balance,omitempty"`
	AvailableBalance *float64  `json:"available_balance,omitempty"`
	CurrencyCode     string    `json:"currency_code"`
	InstitutionName  string    `json:"institution_name"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
