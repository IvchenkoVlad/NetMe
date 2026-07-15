package repositories

import (
	"time"

	"github.com/vladyslavivchenko/netme/internal/models"
)

// ── Auth ──────────────────────────────────────────────────────────────────────

type UserRepo interface {
	CreateUser(email, passwordHash string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	UpdateLastLogin(userID string) error
	FindOrCreateGoogleUser(googleID, email string) (*models.User, error)
	DeleteUser(userID string) error
}

type TokenRepo interface {
	CreateRefreshToken(userID, token string, expiresAt time.Time) (*models.RefreshToken, error)
	GetRefreshToken(token string) (*models.RefreshToken, error)
	RevokeRefreshToken(token, userID string) error
	RevokeAllUserTokens(userID string) error
	IsRefreshTokenValid(token string) (bool, error)
}

// ── Accounts ──────────────────────────────────────────────────────────────────

// AccountLister is the read-only subset needed by AccountsHandler.
type AccountLister interface {
	GetAccountsByUserID(userID string) ([]*models.Account, error)
}

// ── Plaid Items ───────────────────────────────────────────────────────────────

// PlaidItemGetter is the subset of PlaidItemRepository needed by PlaidHandler.
type PlaidItemGetter interface {
	GetItemsByUserID(userID string) ([]*models.PlaidItem, error)
	GetItemByPlaidItemID(plaidItemID string) (*models.PlaidItem, string, error)
}

// ── Transactions ──────────────────────────────────────────────────────────────

// TxnRepo is the subset of TransactionRepository needed by TransactionsHandler.
// Moved here from handlers/transactions.go.
type TxnRepo interface {
	GetTransactionsByUserID(userID, accountID, month string, limit, offset int) ([]*models.Transaction, error)
	GetTransactionByID(userID, id string) (*models.Transaction, error)
	PatchTransactionCategory(userID, txnID, categoryID string) (*models.Transaction, error)
}

// ── Rules ─────────────────────────────────────────────────────────────────────

// RulesRepo is the full interface for RulesRepository, used by RulesHandler.
// Moved here from handlers/rules.go.
type RulesRepo interface {
	Upsert(userID, normalizedMerchant, categoryID string) (*models.CategoryRule, error)
	ApplyToPast(userID, normalizedMerchant, categoryID string) (int64, error)
	List(userID string) ([]*models.CategoryRule, error)
	Delete(userID, id string) error
	ApplyCategoryRules(userID string) error
}

// ── Analytics ─────────────────────────────────────────────────────────────────

// NetWorthReader is the subset of AccountRepository needed by AnalyticsHandler.
type NetWorthReader interface {
	GetNetWorth(userID string) (*models.NetWorth, error)
}

// ── Events ────────────────────────────────────────────────────────────────────

// EventLogger is the subset of EventRepository needed by PlaidHandler.
type EventLogger interface {
	LogRawEvent(userID, eventType string, payload any)
}
