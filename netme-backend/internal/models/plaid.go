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

type Transaction struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	AccountID          string    `json:"account_id"`
	PlaidTransactionID string    `json:"plaid_transaction_id"`
	Amount             float64   `json:"amount"`
	CurrencyCode       string    `json:"currency_code"`
	Name               string    `json:"name"`
	MerchantName       *string   `json:"merchant_name,omitempty"`
	Date               string    `json:"date"`
	AuthorizedDate     *string   `json:"authorized_date,omitempty"`
	Category           *string   `json:"category,omitempty"`
	CategoryDetailed   *string   `json:"category_detailed,omitempty"`
	PaymentChannel     *string   `json:"payment_channel,omitempty"`
	Pending            bool      `json:"pending"`
	CategoryID         *string   `json:"category_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
