package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestAdminOIDCRequiresVerifiedAllowedSubject(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": issuer, "jwks_uri": issuer + "/jwks"})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
				"kty": "RSA", "kid": "test-key", "use": "sig", "alg": "RS256",
				"n": base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
			}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	issuer = provider.URL
	setupContext, cancel := context.WithCancel(context.Background())
	auth, err := NewAdminOIDC(setupContext, issuer+"/.well-known/openid-configuration", issuer, "wishlist-api", "admin-subject")
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privateKey}, (&jose.SignerOptions{}).WithHeader("kid", "test-key"))
	if err != nil {
		t.Fatal(err)
	}
	token := func(subject, audience string, expiry time.Time) string {
		t.Helper()
		raw, err := jwt.Signed(signer).Claims(map[string]any{
			"iss": issuer, "sub": subject, "aud": audience,
			"iat": time.Now().Add(-time.Minute).Unix(), "exp": expiry.Unix(),
		}).Serialize()
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	request := func(raw string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/v1/admin/session", nil)
		r.Header.Set("Authorization", "Bearer "+raw)
		return r
	}
	valid := token("admin-subject", "wishlist-api", time.Now().Add(time.Hour))
	if subject, err := auth.Authorize(request(valid)); err != nil || subject != "admin-subject" {
		t.Fatalf("valid admin rejected: %q %v", subject, err)
	}
	if _, err := auth.Authorize(request(token("other-subject", "wishlist-api", time.Now().Add(time.Hour)))); !errors.Is(err, errAdminForbidden) {
		t.Fatalf("non-admin subject: %v", err)
	}
	for name, raw := range map[string]string{
		"wrong audience":        token("admin-subject", "other-api", time.Now().Add(time.Hour)),
		"expired":               token("admin-subject", "wishlist-api", time.Now().Add(-time.Hour)),
		"old development token": "local-development-admin-token",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := auth.Authorize(request(raw)); !errors.Is(err, errAdminUnauthenticated) {
				t.Fatalf("expected rejection, got %v", err)
			}
		})
	}
	if _, err := NewAdminOIDC(context.Background(), issuer+"/.well-known/openid-configuration", "https://wrong.example", "wishlist-api", "admin-subject"); err == nil {
		t.Fatal("discovery issuer mismatch was accepted")
	}
	t.Setenv("APP_ENV", "production")
	if _, err := NewAdminOIDC(context.Background(), issuer+"/.well-known/openid-configuration", issuer, "wishlist-api", "admin-subject"); err == nil {
		t.Fatal("production accepted HTTP OIDC discovery")
	}
}

func TestProductionAdminNeverFallsBackToDevelopmentToken(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("WISHLIST_ADMIN_TOKEN", "old-token")
	server := New(nil, nil, true)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/session", nil)
	request.Header.Set("Authorization", "Bearer old-token")
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("production token fallback status = %d", response.Code)
	}
}
