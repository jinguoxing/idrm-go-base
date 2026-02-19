package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestOIDCValidatorValidateTokenSuccess(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key failed: %v", err)
	}
	jwks := buildJWKS("k1", &privateKey.PublicKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cfg := Config{
		Mode:        "oidc",
		JWKSURL:     server.URL,
		Issuer:      "https://issuer.example.com",
		Audience:    []string{"api://system-service"},
		UserIDClaim: "sub",
	}
	validator, err := NewValidator(cfg)
	if err != nil {
		t.Fatalf("new validator failed: %v", err)
	}

	tokenString, err := createRS256Token(privateKey, "k1", jwt.MapClaims{
		"sub": "user-123",
		"iss": "https://issuer.example.com",
		"aud": "api://system-service",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign token failed: %v", err)
	}

	claims, err := validator.ValidateToken(context.Background(), tokenString)
	if err != nil {
		t.Fatalf("validate token failed: %v", err)
	}
	if got, _ := claims["sub"].(string); got != "user-123" {
		t.Fatalf("unexpected sub claim: %v", got)
	}
}

func TestOIDCValidatorValidateTokenAudienceMismatch(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key failed: %v", err)
	}
	jwks := buildJWKS("k1", &privateKey.PublicKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cfg := Config{
		Mode:        "oidc",
		JWKSURL:     server.URL,
		Issuer:      "https://issuer.example.com",
		Audience:    []string{"api://expected"},
		UserIDClaim: "sub",
	}
	validator, err := NewValidator(cfg)
	if err != nil {
		t.Fatalf("new validator failed: %v", err)
	}

	tokenString, err := createRS256Token(privateKey, "k1", jwt.MapClaims{
		"sub": "user-123",
		"iss": "https://issuer.example.com",
		"aud": "api://other",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign token failed: %v", err)
	}

	if _, err := validator.ValidateToken(context.Background(), tokenString); err == nil {
		t.Fatal("expected audience mismatch error, got nil")
	}
}

func createRS256Token(privateKey *rsa.PrivateKey, kid string, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(privateKey)
}

func buildJWKS(kid string, publicKey *rsa.PublicKey) map[string]any {
	return map[string]any{
		"keys": []map[string]string{
			{
				"kid": kid,
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   encodeBigInt(publicKey.N),
				"e":   encodeBigInt(big.NewInt(int64(publicKey.E))),
			},
		},
	}
}

func encodeBigInt(v *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(v.Bytes())
}
