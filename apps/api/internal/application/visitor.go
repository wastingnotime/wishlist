package application

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
)

const (
	OTPValidity      = 10 * time.Minute
	SessionValidity  = 30 * 24 * time.Hour
	PrivacyFreshness = 10 * time.Minute
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type OTPChallenge struct {
	ID        string
	Email     string
	Digest    string
	ExpiresAt time.Time
	Attempts  int
}

type PrivacyVote struct {
	FeatureID string    `json:"feature_id"`
	AppName   string    `json:"app_name"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type PrivacySuggestion struct {
	ID                 string     `json:"id"`
	AppName            string     `json:"app_name"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	ReviewedAt         *time.Time `json:"reviewed_at"`
	ResultingFeatureID *string    `json:"resulting_feature_id"`
}

type PrivacyData struct {
	Email          string              `json:"email"`
	CreatedAt      time.Time           `json:"created_at"`
	LastVerifiedAt time.Time           `json:"last_verified_at"`
	ActiveSessions int                 `json:"active_sessions"`
	Votes          []PrivacyVote       `json:"votes"`
	Suggestions    []PrivacySuggestion `json:"suggestions"`
}

type PublicPrivacyReview struct {
	FeatureID   string    `json:"feature_id"`
	AppName     string    `json:"app_name"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	DeliveryURL string    `json:"delivery_url"`
	RequestedAt time.Time `json:"requested_at"`
}

type VisitorStore interface {
	SaveOTPChallenge(context.Context, OTPChallenge, time.Time) (bool, error)
	OTPChallenge(context.Context, string) (OTPChallenge, bool, error)
	RecordOTPFailure(context.Context, string) error
	CompleteOTP(context.Context, string, string, time.Time, string, string, time.Time) (string, error)
	VisitorSession(context.Context, string, time.Time) (string, bool, error)
	DeleteVisitorSession(context.Context, string) error
	VoteFeatureIDs(context.Context, string) ([]string, error)
	ToggleVote(context.Context, string, string, time.Time) (bool, int, error)
	CreateSuggestion(context.Context, string, string, string, string, string, time.Time) (string, error)
	PrivacySession(context.Context, string, time.Time) (string, error)
	PrivacyData(context.Context, string, time.Time) (PrivacyData, error)
	CompleteEmailCorrection(context.Context, string, string, string, time.Time) error
	ErasePrivacyData(context.Context, string, time.Time) error
}

type OTPSender interface {
	Send(context.Context, string, string) error
}

type Clock interface{ Now() time.Time }

type Visitor struct {
	store  VisitorStore
	sender OTPSender
	clock  Clock
	secret []byte
	random io.Reader
}

func NewVisitor(store VisitorStore, sender OTPSender, clock Clock, secret string) *Visitor {
	return &Visitor{store: store, sender: sender, clock: clock, secret: []byte(secret), random: rand.Reader}
}

func (visitor *Visitor) RequestOTP(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) > 254 || !emailPattern.MatchString(email) {
		return domain.ErrInvalidRequest
	}
	challengeID, err := randomToken(visitor.random, 16)
	if err != nil {
		return err
	}
	codeNumber, err := rand.Int(visitor.random, big.NewInt(1_000_000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%06d", codeNumber.Int64())
	now := visitor.clock.Now()
	challenge := OTPChallenge{ID: challengeID, Email: email, Digest: visitor.digest(challengeID, code), ExpiresAt: now.Add(OTPValidity)}
	allowed, err := visitor.store.SaveOTPChallenge(ctx, challenge, now)
	if err != nil {
		return err
	}
	if !allowed {
		return nil
	}
	if err := visitor.sender.Send(ctx, email, code); err != nil {
		return err
	}
	return nil
}

func (visitor *Visitor) VerifyOTP(ctx context.Context, email, code string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) > 254 || !emailPattern.MatchString(email) || len(code) != 6 || !regexp.MustCompile(`^[0-9]{6}$`).MatchString(code) {
		return "", domain.ErrInvalidCode
	}
	now := visitor.clock.Now()
	challenge, found, err := visitor.store.OTPChallenge(ctx, email)
	if err != nil {
		return "", err
	}
	if !found || !now.Before(challenge.ExpiresAt) || challenge.Attempts >= 5 {
		return "", domain.ErrInvalidCode
	}
	if !hmac.Equal([]byte(challenge.Digest), []byte(visitor.digest(challenge.ID, code))) {
		if err := visitor.store.RecordOTPFailure(ctx, email); err != nil {
			return "", err
		}
		return "", domain.ErrInvalidCode
	}
	sessionID, err := randomToken(visitor.random, 32)
	if err != nil {
		return "", err
	}
	identityID, err := randomToken(visitor.random, 16)
	if err != nil {
		return "", err
	}
	_, err = visitor.store.CompleteOTP(ctx, email, challenge.ID, now, identityID, sessionID, now.Add(SessionValidity))
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func (visitor *Visitor) Session(ctx context.Context, sessionID string) ([]string, error) {
	if sessionID == "" {
		return nil, domain.ErrUnauthenticated
	}
	now := visitor.clock.Now()
	identityID, found, err := visitor.store.VisitorSession(ctx, sessionID, now)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, domain.ErrUnauthenticated
	}
	return visitor.store.VoteFeatureIDs(ctx, identityID)
}

func (visitor *Visitor) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return visitor.store.DeleteVisitorSession(ctx, sessionID)
}

func (visitor *Visitor) ToggleVote(ctx context.Context, sessionID, featureID string) (bool, int, error) {
	if sessionID == "" {
		return false, 0, domain.ErrUnauthenticated
	}
	identityID, found, err := visitor.store.VisitorSession(ctx, sessionID, visitor.clock.Now())
	if err != nil {
		return false, 0, err
	}
	if !found {
		return false, 0, domain.ErrUnauthenticated
	}
	return visitor.store.ToggleVote(ctx, identityID, featureID, visitor.clock.Now())
}

func (visitor *Visitor) SubmitSuggestion(ctx context.Context, sessionID, appID, title, description string) (string, error) {
	if sessionID == "" {
		return "", domain.ErrUnauthenticated
	}
	identityID, found, err := visitor.store.VisitorSession(ctx, sessionID, visitor.clock.Now())
	if err != nil {
		return "", err
	}
	if !found {
		return "", domain.ErrUnauthenticated
	}
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if len(title) == 0 || len(title) > 160 || len(description) > 2000 {
		return "", domain.ErrInvalidRequest
	}
	id, err := randomToken(visitor.random, 16)
	if err != nil {
		return "", err
	}
	return visitor.store.CreateSuggestion(ctx, id, appID, identityID, title, description, visitor.clock.Now())
}

func (visitor *Visitor) PrivacyData(ctx context.Context, sessionID string) (PrivacyData, error) {
	if sessionID == "" {
		return PrivacyData{}, domain.ErrUnauthenticated
	}
	return visitor.store.PrivacyData(ctx, sessionID, visitor.clock.Now())
}

func (visitor *Visitor) RequestEmailCorrection(ctx context.Context, sessionID, newEmail string) error {
	if sessionID == "" {
		return domain.ErrUnauthenticated
	}
	current, err := visitor.store.PrivacySession(ctx, sessionID, visitor.clock.Now())
	if err != nil {
		return err
	}
	newEmail = strings.ToLower(strings.TrimSpace(newEmail))
	if newEmail == current || len(newEmail) > 254 || !emailPattern.MatchString(newEmail) {
		return domain.ErrInvalidRequest
	}
	return visitor.RequestOTP(ctx, newEmail)
}

func (visitor *Visitor) CompleteEmailCorrection(ctx context.Context, sessionID, newEmail, code string) error {
	if sessionID == "" {
		return domain.ErrUnauthenticated
	}
	newEmail = strings.ToLower(strings.TrimSpace(newEmail))
	if len(newEmail) > 254 || !emailPattern.MatchString(newEmail) || len(code) != 6 || !regexp.MustCompile(`^[0-9]{6}$`).MatchString(code) {
		return domain.ErrInvalidCode
	}
	now := visitor.clock.Now()
	if _, err := visitor.store.PrivacySession(ctx, sessionID, now); err != nil {
		return err
	}
	challenge, found, err := visitor.store.OTPChallenge(ctx, newEmail)
	if err != nil {
		return err
	}
	if !found || !now.Before(challenge.ExpiresAt) || challenge.Attempts >= 5 {
		return domain.ErrInvalidCode
	}
	if !hmac.Equal([]byte(challenge.Digest), []byte(visitor.digest(challenge.ID, code))) {
		if err := visitor.store.RecordOTPFailure(ctx, newEmail); err != nil {
			return err
		}
		return domain.ErrInvalidCode
	}
	return visitor.store.CompleteEmailCorrection(ctx, sessionID, newEmail, challenge.ID, now)
}

func (visitor *Visitor) ErasePrivacyData(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return domain.ErrUnauthenticated
	}
	return visitor.store.ErasePrivacyData(ctx, sessionID, visitor.clock.Now())
}

func (visitor *Visitor) digest(challengeID, code string) string {
	mac := hmac.New(sha256.New, visitor.secret)
	_, _ = mac.Write([]byte(challengeID + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomToken(source io.Reader, size int) (string, error) {
	data := make([]byte, size)
	if _, err := io.ReadFull(source, data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}
