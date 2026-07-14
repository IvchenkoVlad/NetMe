package models

import "time"

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
