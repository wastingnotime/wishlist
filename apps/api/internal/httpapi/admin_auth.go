package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

var errAdminUnauthenticated = errors.New("admin authentication required")
var errAdminForbidden = errors.New("admin subject is not allowed")

type AdminOIDC struct {
	verifier *oidc.IDTokenVerifier
	subjects map[string]struct{}
}

// NewAdminOIDC pins discovery to the configured issuer and grants admin access
// only to explicitly listed Casdoor subjects. Email and display names are not
// authority identifiers.
func NewAdminOIDC(ctx context.Context, discoveryURL, issuer, audience, subjects string) (*AdminOIDC, error) {
	if discoveryURL == "" || issuer == "" || audience == "" || strings.TrimSpace(subjects) == "" {
		return nil, errors.New("OIDC discovery URL, issuer, audience, and admin subjects are required")
	}
	if os.Getenv("APP_ENV") == "production" && (!isHTTPS(discoveryURL) || !isHTTPS(issuer)) {
		return nil, errors.New("production OIDC discovery and issuer must use HTTPS")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("load OIDC discovery: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("load OIDC discovery: status %d", response.StatusCode)
	}
	var metadata struct {
		Issuer  string `json:"issuer"`
		JWKSURL string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return nil, err
	}
	if metadata.Issuer != issuer || metadata.JWKSURL == "" {
		return nil, errors.New("OIDC discovery issuer mismatch or missing JWKS URL")
	}
	if os.Getenv("APP_ENV") == "production" && !isHTTPS(metadata.JWKSURL) {
		return nil, errors.New("production OIDC JWKS URL must use HTTPS")
	}
	allowed := map[string]struct{}{}
	for _, subject := range strings.Split(subjects, ",") {
		if subject = strings.TrimSpace(subject); subject != "" {
			allowed[subject] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return nil, errors.New("at least one admin subject is required")
	}
	keyContext := oidc.ClientContext(context.Background(), &http.Client{Timeout: 10 * time.Second})
	keySet := oidc.NewRemoteKeySet(keyContext, metadata.JWKSURL)
	return &AdminOIDC{verifier: oidc.NewVerifier(issuer, keySet, &oidc.Config{ClientID: audience}), subjects: allowed}, nil
}

func isHTTPS(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func (a *AdminOIDC) Authorize(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", errAdminUnauthenticated
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if raw == "" {
		return "", errAdminUnauthenticated
	}
	token, err := a.verifier.Verify(r.Context(), raw)
	if err != nil {
		return "", errAdminUnauthenticated
	}
	// The verifier checks issuer, signature, expiry, and configured audience.
	subject := strings.TrimSpace(token.Subject)
	if subject == "" {
		return "", errAdminUnauthenticated
	}
	if _, ok := a.subjects[subject]; !ok {
		return "", errAdminForbidden
	}
	return subject, nil
}
