package repositories

import (
	"database/sql"
	"errors"
	"time"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type TokenRepository struct {
	db *sql.DB
}

func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) CreateRefreshToken(userID, token string, expiresAt time.Time) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{}
	err := r.db.QueryRow(
		`INSERT INTO refresh_tokens (user_id, token, expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, now(), now())
		 RETURNING id, user_id, token, expires_at, revoked_at, created_at, updated_at`,
		userID, token, expiresAt,
	).Scan(
		&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt,
		&rt.RevokedAt, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (r *TokenRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{}
	err := r.db.QueryRow(
		`SELECT id, user_id, token, expires_at, revoked_at, created_at, updated_at
		 FROM refresh_tokens WHERE token = $1`,
		token,
	).Scan(
		&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt,
		&rt.RevokedAt, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("refresh token not found")
		}
		return nil, err
	}
	return rt, nil
}

func (r *TokenRepository) RevokeRefreshToken(token, userID string) error {
	_, err := r.db.Exec(
		`UPDATE refresh_tokens SET revoked_at = now(), updated_at = now() WHERE token = $1 AND user_id = $2`,
		token, userID,
	)
	return err
}

func (r *TokenRepository) RevokeAllUserTokens(userID string) error {
	_, err := r.db.Exec(
		`UPDATE refresh_tokens SET revoked_at = now(), updated_at = now() WHERE user_id = $1`,
		userID,
	)
	return err
}

func (r *TokenRepository) IsRefreshTokenValid(token string) (bool, error) {
	rt, err := r.GetRefreshToken(token)
	if err != nil {
		return false, err
	}
	if rt.RevokedAt != nil {
		return false, errors.New("token is revoked")
	}
	if time.Now().After(rt.ExpiresAt) {
		return false, errors.New("token is expired")
	}
	return true, nil
}
