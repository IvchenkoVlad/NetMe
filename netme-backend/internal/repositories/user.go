package repositories

import (
	"database/sql"
	"errors"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(email, passwordHash string) (*models.User, error) {
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

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
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

func (r *UserRepository) GetUserByID(userID string) (*models.User, error) {
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

func (r *UserRepository) FindOrCreateGoogleUser(googleID, email string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		`SELECT id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at
		 FROM users WHERE auth_provider = 'google' AND auth_provider_user_id = $1`,
		googleID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.AuthProvider, &user.AuthProviderUserID, &user.CreatedAt, &user.UpdatedAt)
	if err == nil {
		return user, nil
	}

	err = r.db.QueryRow(
		`INSERT INTO users (email, auth_provider, auth_provider_user_id, created_at, updated_at)
		 VALUES ($1, 'google', $2, now(), now())
		 ON CONFLICT (email) DO UPDATE SET auth_provider_user_id = $2, updated_at = now()
		 RETURNING id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at`,
		email, googleID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.AuthProvider, &user.AuthProviderUserID, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *UserRepository) UpdateLastLogin(userID string) error {
	_, err := r.db.Exec(
		`UPDATE users SET updated_at = now() WHERE id = $1`,
		userID,
	)
	return err
}
