package repositories

import (
	"database/sql"
	"errors"
	"time"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CreateUser(email, passwordHash string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		`INSERT INTO users (email, password_hash, auth_provider, created_at, updated_at)
		 VALUES ($1, $2, 'local', now(), now())
		 RETURNING id, email, auth_provider, auth_provider_user_id, created_at, updated_at`,
		email, passwordHash,
	).Scan(
		&user.ID, &user.Email, &user.AuthProvider,
		&user.AuthProviderUserID, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = passwordHash
	return user, nil
}

func (r *AuthRepository) GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		`SELECT id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.AuthProvider, &user.AuthProviderUserID,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) GetUserByID(userID string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		`SELECT id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at
		 FROM users WHERE id = $1`,
		userID,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.AuthProvider, &user.AuthProviderUserID,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) UpdateLastLogin(userID string) error {
	_, err := r.db.Exec(
		`UPDATE users SET updated_at = now() WHERE id = $1`,
		userID,
	)
	return err
}

func (r *AuthRepository) CreateRefreshToken(userID, token string, expiresAt time.Time) (*models.RefreshToken, error) {
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

func (r *AuthRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
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

func (r *AuthRepository) RevokeRefreshToken(token string) error {
	_, err := r.db.Exec(
		`UPDATE refresh_tokens SET revoked_at = now(), updated_at = now() WHERE token = $1`,
		token,
	)
	return err
}

func (r *AuthRepository) RevokeAllUserTokens(userID string) error {
	_, err := r.db.Exec(
		`UPDATE refresh_tokens SET revoked_at = now(), updated_at = now() WHERE user_id = $1`,
		userID,
	)
	return err
}

func (r *AuthRepository) IsRefreshTokenValid(token string) (bool, error) {
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
