package repositories

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
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
			return nil, ErrUserNotFound
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
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) FindOrCreateGoogleUser(googleID, email string) (*models.User, error) {
	user := &models.User{}

	// Step 1: existing Google user — normal login path
	err := r.db.QueryRow(
		`SELECT id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at
		 FROM users WHERE auth_provider = 'google' AND auth_provider_user_id = $1`,
		googleID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.AuthProvider,
		&user.AuthProviderUserID, &user.CreatedAt, &user.UpdatedAt)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Step 2: email exists with a different provider — refuse, prevent account takeover
	var existingProvider string
	err = r.db.QueryRow(`SELECT auth_provider FROM users WHERE email = $1`, email).Scan(&existingProvider)
	if err == nil {
		return nil, ErrEmailTakenByOtherProvider
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Step 3: new user — create Google account
	user = &models.User{}
	err = r.db.QueryRow(
		`INSERT INTO users (email, auth_provider, auth_provider_user_id, created_at, updated_at)
		 VALUES ($1, 'google', $2, now(), now())
		 RETURNING id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at`,
		email, googleID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.AuthProvider,
		&user.AuthProviderUserID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrEmailTakenByOtherProvider
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) UpdateLastLogin(userID string) error {
	_, err := r.db.Exec(
		`UPDATE users SET updated_at = now() WHERE id = $1`,
		userID,
	)
	return err
}

func (r *UserRepository) DeleteUser(userID string) error {
	result, err := r.db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}
