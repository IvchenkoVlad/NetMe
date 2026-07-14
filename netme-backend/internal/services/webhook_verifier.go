package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	plaid "github.com/plaid/plaid-go/v42/plaid"
)

// WebhookVerifier verifies Plaid webhook signatures using ES256 JWTs.
// Keys are fetched from Plaid on first use and cached in memory.
type WebhookVerifier struct {
	client   *plaid.APIClient
	env      string
	keyCache sync.Map // kid -> *ecdsa.PublicKey
}

func NewWebhookVerifier(client *plaid.APIClient, env string) *WebhookVerifier {
	return &WebhookVerifier{client: client, env: env}
}

type webhookClaims struct {
	RequestBodySHA256 string `json:"request_body_sha256"`
	jwt.RegisteredClaims
}

// Verify checks the Plaid-Verification JWT header against the raw request body.
// Returns nil if the signature and body hash are both valid.
// In non-production environments verification is skipped.
func (v *WebhookVerifier) Verify(tokenStr string, body []byte) error {
	if v.env != "production" {
		return nil
	}
	if tokenStr == "" {
		return errors.New("missing Plaid-Verification header")
	}

	// Parse unverified to extract key ID from JWT header.
	unverified, _, err := new(jwt.Parser).ParseUnverified(tokenStr, &webhookClaims{})
	if err != nil {
		return fmt.Errorf("parse webhook token: %w", err)
	}
	kid, ok := unverified.Header["kid"].(string)
	if !ok || kid == "" {
		return errors.New("webhook token missing kid")
	}

	pub, err := v.getKey(kid)
	if err != nil {
		return err
	}

	verified, err := jwt.ParseWithClaims(tokenStr, &webhookClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pub, nil
	})
	if err != nil || !verified.Valid {
		return fmt.Errorf("webhook signature invalid: %w", err)
	}

	claims, ok := verified.Claims.(*webhookClaims)
	if !ok {
		return errors.New("unexpected claims type")
	}

	// Verify body hash.
	sum := sha256.Sum256(body)
	expected := fmt.Sprintf("%x", sum[:])
	if claims.RequestBodySHA256 != expected {
		return errors.New("request body hash mismatch")
	}

	return nil
}

func (v *WebhookVerifier) getKey(kid string) (*ecdsa.PublicKey, error) {
	if cached, ok := v.keyCache.Load(kid); ok {
		return cached.(*ecdsa.PublicKey), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := plaid.NewWebhookVerificationKeyGetRequest(kid)
	resp, _, err := v.client.PlaidApi.WebhookVerificationKeyGet(ctx).
		WebhookVerificationKeyGetRequest(*req).Execute()
	if err != nil {
		return nil, fmt.Errorf("fetch webhook key: %w", err)
	}

	key := resp.GetKey()
	pub, err := ecdsaFromJWK(key.GetX(), key.GetY())
	if err != nil {
		return nil, err
	}

	v.keyCache.Store(kid, pub)
	return pub, nil
}

func ecdsaFromJWK(xB64, yB64 string) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(xB64)
	if err != nil {
		return nil, fmt.Errorf("decode JWK x: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(yB64)
	if err != nil {
		return nil, fmt.Errorf("decode JWK y: %w", err)
	}
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}
