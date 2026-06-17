package repositories

import (
	"time"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type UserRepo interface {
	CreateUser(email, passwordHash string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	UpdateLastLogin(userID string) error
}

type TokenRepo interface {
	CreateRefreshToken(userID, token string, expiresAt time.Time) (*models.RefreshToken, error)
	GetRefreshToken(token string) (*models.RefreshToken, error)
	RevokeRefreshToken(token, userID string) error
	RevokeAllUserTokens(userID string) error
	IsRefreshTokenValid(token string) (bool, error)
}
