package models

import "time"

type CategoryRule struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	NormalizedMerchant string    `json:"normalized_merchant"`
	CategoryID         string    `json:"category_id"`
	Category           *Category `json:"category,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
