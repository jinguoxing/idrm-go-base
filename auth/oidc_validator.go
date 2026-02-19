package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	defaultJWKSRequestTimeout = 3 * time.Second
	defaultJWKSCacheTTL       = 5 * time.Minute
)

// oidcValidator 使用 JWKS 验签 OIDC/JWT（当前支持 RSA 系列算法）。
type oidcValidator struct {
	cfg Config

	client *http.Client

	mu       sync.RWMutex
	keySet   map[string]crypto.PublicKey
	expires  time.Time
	cacheTTL time.Duration
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func newOIDCValidator(cfg Config) (Validator, error) {
	if strings.TrimSpace(cfg.JWKSURL) == "" {
		return nil, fmt.Errorf("auth: JWKSURL required when mode=oidc")
	}

	return &oidcValidator{
		cfg: cfg,
		client: &http.Client{
			Timeout: defaultJWKSRequestTimeout,
		},
		keySet:   make(map[string]crypto.PublicKey),
		cacheTTL: defaultJWKSCacheTTL,
	}, nil
}

func (v *oidcValidator) ValidateToken(ctx context.Context, tokenString string) (map[string]interface{}, error) {
	parser := jwt.Parser{
		ValidMethods: []string{"RS256", "RS384", "RS512"},
	}
	claims := jwt.MapClaims{}

	token, err := parser.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		return v.getVerificationKey(ctx, kid)
	})
	if err != nil {
		return nil, err
	}
	if token == nil || !token.Valid {
		return nil, fmt.Errorf("auth: invalid oidc token")
	}

	if err := v.validateIssuer(claims); err != nil {
		return nil, err
	}
	if err := v.validateAudience(claims); err != nil {
		return nil, err
	}

	out := make(map[string]interface{}, len(claims))
	for k, val := range claims {
		out[k] = val
	}
	return out, nil
}

func (v *oidcValidator) getVerificationKey(ctx context.Context, kid string) (crypto.PublicKey, error) {
	keys, err := v.getCachedKeys(ctx, false)
	if err != nil {
		return nil, err
	}
	if key, ok := pickJWKKey(keys, kid); ok {
		return key, nil
	}

	// kid 缺失或发生轮转时，强制刷新一次。
	keys, err = v.getCachedKeys(ctx, true)
	if err != nil {
		return nil, err
	}
	if key, ok := pickJWKKey(keys, kid); ok {
		return key, nil
	}

	if kid == "" {
		return nil, fmt.Errorf("auth: no jwk key available for token without kid")
	}
	return nil, fmt.Errorf("auth: jwk key not found for kid=%s", kid)
}

func (v *oidcValidator) getCachedKeys(ctx context.Context, forceRefresh bool) (map[string]crypto.PublicKey, error) {
	now := time.Now()
	if !forceRefresh {
		v.mu.RLock()
		if len(v.keySet) > 0 && now.Before(v.expires) {
			keys := cloneKeyMap(v.keySet)
			v.mu.RUnlock()
			return keys, nil
		}
		v.mu.RUnlock()
	}

	keys, err := v.fetchJWKS(ctx)
	if err != nil {
		return nil, err
	}

	v.mu.Lock()
	v.keySet = keys
	v.expires = time.Now().Add(v.cacheTTL)
	keys = cloneKeyMap(v.keySet)
	v.mu.Unlock()

	return keys, nil
}

func (v *oidcValidator) fetchJWKS(ctx context.Context) (map[string]crypto.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("auth: build jwks request failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth: fetch jwks failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("auth: fetch jwks status=%d", resp.StatusCode)
	}

	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("auth: decode jwks failed: %w", err)
	}
	if len(doc.Keys) == 0 {
		return nil, fmt.Errorf("auth: jwks contains no keys")
	}

	keys := make(map[string]crypto.PublicKey, len(doc.Keys))
	fallbackIndex := 0
	for _, jwk := range doc.Keys {
		pub, err := parseJWKPublicKey(jwk)
		if err != nil {
			continue
		}
		kid := strings.TrimSpace(jwk.Kid)
		if kid == "" {
			kid = fmt.Sprintf("_anon_%d", fallbackIndex)
			fallbackIndex++
		}
		keys[kid] = pub
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("auth: no supported jwk public keys")
	}
	return keys, nil
}

func parseJWKPublicKey(jwk jwkKey) (crypto.PublicKey, error) {
	if strings.TrimSpace(jwk.Kty) != "RSA" {
		return nil, fmt.Errorf("unsupported kty=%s", jwk.Kty)
	}
	n, err := decodeBase64URLBigInt(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("decode rsa modulus failed: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("decode rsa exponent failed: %w", err)
	}
	if len(eBytes) == 0 {
		return nil, fmt.Errorf("empty rsa exponent")
	}

	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if e <= 1 {
		return nil, fmt.Errorf("invalid rsa exponent")
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

func decodeBase64URLBigInt(encoded string) (*big.Int, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("empty value")
	}

	n := new(big.Int).SetBytes(bytes)
	if n.Sign() <= 0 {
		return nil, fmt.Errorf("invalid bigint value")
	}
	return n, nil
}

func pickJWKKey(keys map[string]crypto.PublicKey, kid string) (crypto.PublicKey, bool) {
	if len(keys) == 0 {
		return nil, false
	}
	if strings.TrimSpace(kid) != "" {
		key, ok := keys[kid]
		return key, ok
	}
	if len(keys) == 1 {
		for _, key := range keys {
			return key, true
		}
	}
	return nil, false
}

func cloneKeyMap(src map[string]crypto.PublicKey) map[string]crypto.PublicKey {
	dst := make(map[string]crypto.PublicKey, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (v *oidcValidator) validateIssuer(claims jwt.MapClaims) error {
	issuer := strings.TrimSpace(v.cfg.Issuer)
	if issuer == "" {
		return nil
	}
	iss, _ := claims["iss"].(string)
	if iss != issuer {
		return fmt.Errorf("auth: issuer mismatch expected=%s actual=%s", issuer, iss)
	}
	return nil
}

func (v *oidcValidator) validateAudience(claims jwt.MapClaims) error {
	if len(v.cfg.Audience) == 0 {
		return nil
	}

	var tokenAud []string
	switch aud := claims["aud"].(type) {
	case string:
		if strings.TrimSpace(aud) != "" {
			tokenAud = []string{aud}
		}
	case []string:
		tokenAud = aud
	case []interface{}:
		for _, item := range aud {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				tokenAud = append(tokenAud, s)
			}
		}
	}
	if len(tokenAud) == 0 {
		return fmt.Errorf("auth: audience claim missing")
	}

	want := make(map[string]struct{}, len(v.cfg.Audience))
	for _, aud := range v.cfg.Audience {
		if trimmed := strings.TrimSpace(aud); trimmed != "" {
			want[trimmed] = struct{}{}
		}
	}
	if len(want) == 0 {
		return nil
	}

	for _, aud := range tokenAud {
		if _, ok := want[aud]; ok {
			return nil
		}
	}
	return fmt.Errorf("auth: audience mismatch expected_one_of=%v actual=%v", v.cfg.Audience, tokenAud)
}
