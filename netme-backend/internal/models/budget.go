package models

import "time"

type Category struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	Name                  string    `json:"name"`
	Icon                  string    `json:"icon"`
	Color                 string    `json:"color"`
	IsIncome              bool      `json:"is_income"`
	SortOrder             int       `json:"sort_order"`
	PlaidPrimaryCategories []string `json:"plaid_primary_categories"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type Budget struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	CategoryID string    `json:"category_id"`
	Month      string    `json:"month"`
	Amount     float64   `json:"amount"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CategorySummary struct {
	Category
	Spent             float64 `json:"spent"`
	BudgetLimit       float64 `json:"budget_limit"`
	TransactionCount  int     `json:"transaction_count"`
}

type MonthlyTotal struct {
	Month    string  `json:"month"`
	Spending float64 `json:"spending"`
	Income   float64 `json:"income"`
}

type BudgetSummary struct {
	Month      string            `json:"month"`
	Income     float64           `json:"income"`
	Spending   float64           `json:"spending"`
	Categories []CategorySummary `json:"categories"`
}

var DefaultCategories = []struct {
	Name  string
	Icon  string
	Color string
	Income bool
	Order int
	Plaid []string
}{
	{"Housing",        "🏠", "#3b82f6", false, 1,  []string{"RENT_AND_UTILITIES", "HOME_IMPROVEMENT"}},
	{"Food & Dining",  "🍔", "#f59e0b", false, 2,  []string{"FOOD_AND_DRINK"}},
	{"Transport",      "🚗", "#10b981", false, 3,  []string{"TRANSPORTATION"}},
	{"Shopping",       "🛍️", "#8b5cf6", false, 4,  []string{"GENERAL_MERCHANDISE", "PERSONAL_CARE"}},
	{"Entertainment",  "🎬", "#ec4899", false, 5,  []string{"ENTERTAINMENT"}},
	{"Health",         "💊", "#ef4444", false, 6,  []string{"MEDICAL"}},
	{"Travel",         "✈️", "#06b6d4", false, 7,  []string{"TRAVEL"}},
	{"Bills",          "💡", "#64748b", false, 8,  []string{"GENERAL_SERVICES", "GOVERNMENT_AND_NON_PROFIT", "BANK_FEES"}},
	{"Income",         "💰", "#22c55e", true,  9,  []string{"INCOME", "TRANSFER_IN"}},
	{"Transfers",      "🔄", "#94a3b8", false, 10, []string{"TRANSFER_OUT", "LOAN_PAYMENTS"}},
	{"Other",          "📦", "#cbd5e1", false, 11, []string{}},
}
