package services

import (
	"context"
	"errors"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

const refreshTokenTTL = 7 * 24 * time.Hour

var (
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNoPassword         = errors.New("no password set for this account")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type AuthService struct {
	userRepo       repositories.UserRepo
	tokenRepo      repositories.TokenRepo
	jwtSvc         *JWTService
	passwordSvc    *PasswordService
	googleVerifier GoogleVerifier
}

func NewAuthService(
	userRepo repositories.UserRepo,
	tokenRepo repositories.TokenRepo,
	jwtSvc *JWTService,
	googleVerifier GoogleVerifier,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		jwtSvc:         jwtSvc,
		passwordSvc:    NewPasswordService(),
		googleVerifier: googleVerifier,
	}
}

func (s *AuthService) Register(email, password string) (*models.AuthResponse, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, ErrInvalidEmail
	}

	if existing, _ := s.userRepo.GetUserByEmail(email); existing != nil {
		return nil, ErrUserExists
	}

	hash, err := s.passwordSvc.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.CreateUser(email, hash)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrUserExists
		}
		return nil, err
	}

	return s.buildResponse(user)
}

func (s *AuthService) Login(email, password string) (*models.AuthResponse, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, ErrInvalidEmail
	}

	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.PasswordHash == "" {
		return nil, ErrNoPassword
	}

	if err := s.passwordSvc.VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		slog.Warn("failed to update last login", "user_id", user.ID, "error", err)
	}

	return s.buildResponse(user)
}

func (s *AuthService) Refresh(refreshToken string) (*models.AuthResponse, error) {
	valid, err := s.tokenRepo.IsRefreshTokenValid(refreshToken)
	if !valid || err != nil {
		return nil, ErrInvalidToken
	}

	record, err := s.tokenRepo.GetRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.GetUserByID(record.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	resp, err := s.buildResponse(user)
	if err != nil {
		return nil, err
	}

	// Revoke old token after new one is issued — failure is non-fatal, old token expires naturally
	if err := s.tokenRepo.RevokeRefreshToken(refreshToken, user.ID); err != nil {
		slog.Warn("failed to revoke old refresh token during rotation", "user_id", user.ID, "error", err)
	}

	return resp, nil
}

func (s *AuthService) Logout(refreshToken, userID string) error {
	return s.tokenRepo.RevokeRefreshToken(refreshToken, userID)
}

func (s *AuthService) GoogleAuth(ctx context.Context, idToken string) (*models.AuthResponse, error) {
	googleID, email, err := s.googleVerifier.Validate(ctx, idToken, "")
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.FindOrCreateGoogleUser(googleID, email)
	if err != nil {
		return nil, err
	}

	return s.buildResponse(user)
}

func (s *AuthService) buildResponse(user *models.User) (*models.AuthResponse, error) {
	accessToken, err := s.jwtSvc.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshTokenStr, err := s.jwtSvc.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	rt, err := s.tokenRepo.CreateRefreshToken(user.ID, refreshTokenStr, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rt.Token,
		ExpiresIn:    900,
		User:         user,
	}, nil
}

func normalizeEmail(email string) (string, error) {
	parsed, err := mail.ParseAddress(strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return "", err
	}
	return parsed.Address, nil
}
