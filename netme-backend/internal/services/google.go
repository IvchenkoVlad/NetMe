package services

import (
	"context"
	"fmt"

	"google.golang.org/api/idtoken"
)

// GoogleVerifier abstracts Google ID token verification for testability.
type GoogleVerifier interface {
	Validate(ctx context.Context, idToken, audience string) (googleID string, email string, err error)
}

// GoogleIDTokenVerifier verifies Google ID tokens using Google's public keys.
type GoogleIDTokenVerifier struct {
	clientID string
}

func NewGoogleIDTokenVerifier(clientID string) *GoogleIDTokenVerifier {
	return &GoogleIDTokenVerifier{clientID: clientID}
}

func (v *GoogleIDTokenVerifier) Validate(ctx context.Context, idToken, audience string) (string, string, error) {
	payload, err := idtoken.Validate(ctx, idToken, audience)
	if err != nil {
		return "", "", err
	}
	email, _ := payload.Claims["email"].(string)
	if email == "" {
		return "", "", fmt.Errorf("email not present in Google ID token")
	}
	return payload.Subject, email, nil
}
