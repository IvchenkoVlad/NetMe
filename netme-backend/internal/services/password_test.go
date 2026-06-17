package services_test

import (
	"testing"

	"github.com/vladyslavivchenko/netme/internal/services"
)

func TestHashAndVerifyPassword(t *testing.T) {
	svc := services.NewPasswordService()

	hash, err := svc.HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "mypassword123" {
		t.Fatal("hash must not equal plaintext password")
	}

	if err := svc.VerifyPassword(hash, "mypassword123"); err != nil {
		t.Errorf("expected correct password to verify, got %v", err)
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	svc := services.NewPasswordService()

	hash, _ := svc.HashPassword("correctpassword")
	if err := svc.VerifyPassword(hash, "wrongpassword"); err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestHashesAreUnique(t *testing.T) {
	svc := services.NewPasswordService()

	hash1, _ := svc.HashPassword("samepassword")
	hash2, _ := svc.HashPassword("samepassword")

	if hash1 == hash2 {
		t.Error("expected unique hashes for same password (bcrypt salt), got identical hashes")
	}
}
