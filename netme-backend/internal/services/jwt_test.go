package services_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vladyslavivchenko/netme/internal/services"
)

const testSecret = "test-secret-key-32-chars-minimum!!"

func TestGenerateAndVerifyAccessToken(t *testing.T) {
	svc := services.NewJWTService(testSecret)

	token, err := svc.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	claims, err := svc.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("expected no error verifying token, got %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got %q", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected Email 'test@example.com', got %q", claims.Email)
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	svc := services.NewJWTService(testSecret)

	claims := services.JWTClaims{
		UserID: "user-123",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))

	_, err := svc.VerifyAccessToken(tokenString)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestVerifyTamperedToken(t *testing.T) {
	svc := services.NewJWTService(testSecret)

	token, _ := svc.GenerateAccessToken("user-123", "test@example.com")
	_, err := svc.VerifyAccessToken(token + "tampered")
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestVerifyTokenWrongSecret(t *testing.T) {
	svc1 := services.NewJWTService(testSecret)
	svc2 := services.NewJWTService("different-secret-key-32-chars!!!")

	token, _ := svc1.GenerateAccessToken("user-123", "test@example.com")
	_, err := svc2.VerifyAccessToken(token)
	if err == nil {
		t.Fatal("expected error verifying token with wrong secret, got nil")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	svc := services.NewJWTService(testSecret)

	token1, err := svc.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	token2, _ := svc.GenerateRefreshToken()

	if len(token1) != 64 {
		t.Errorf("expected 64-char hex token, got length %d", len(token1))
	}
	if token1 == token2 {
		t.Error("expected unique refresh tokens, got identical tokens")
	}
}
