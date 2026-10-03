package infrastructure

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
)

func retentionTestStore(t *testing.T) *PostgresStore {
	t.Helper()
	databaseURL := os.Getenv("WISHLIST_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set WISHLIST_TEST_DATABASE_URL to run PostgreSQL retention tests")
	}
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	schema := "wishlist_retention_" + hex.EncodeToString(random)
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	board, err := SampleBoard()
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenPostgres(parsed.String(), board)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
		_, _ = admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
		_ = admin.Close()
	})
	return store
}

func TestRetentionExpiresVotesAndTechnicalRowsWithoutRemovingActiveLinks(t *testing.T) {
	store := retentionTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(-1, 0, -1)
	recent := now.AddDate(-1, 0, 1)
	for _, identity := range []struct {
		id, email string
		verified  time.Time
	}{
		{"old-with-suggestion", "old-suggestion@example.com", old},
		{"old-orphan", "old-orphan@example.com", old},
		{"recent-voter", "recent@example.com", recent},
	} {
		if _, err := store.db.ExecContext(ctx, `INSERT INTO identities(id,email,created_at,last_verified_at) VALUES($1,$2,$3,$4)`, identity.id, identity.email, old, identity.verified); err != nil {
			t.Fatal(err)
		}
	}
	for _, identity := range []string{"old-with-suggestion", "recent-voter"} {
		if _, err := store.db.ExecContext(ctx, `INSERT INTO votes(identity_id,feature_id,created_at) VALUES($1,'feature-family-sharing',$2)`, identity, old); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO suggestions(id,app_id,identity_id,title,description,status,created_at,reviewed_at) VALUES('kept','app-cat-care','old-with-suggestion','idea','detail','rejected',$1,$2)`, old, now.AddDate(0, -1, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO otp_challenges(id,email,digest,expires_at) VALUES('expired','old@example.com','digest',$1)`, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO otp_requests(email,requested_at) VALUES('old@example.com',$1)`, now.Add(-25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO otp_requests(email,requested_at) VALUES('recent@example.com',$1)`, now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO sessions(id,identity_id,expires_at) VALUES('expired','recent-voter',$1)`, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	counts, err := store.RunRetention(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if counts.OTPChallenges != 1 || counts.OTPRequests != 1 || counts.Sessions != 1 || counts.Votes != 1 || counts.Identities != 1 || counts.Suggestions != 0 {
		t.Fatalf("unexpected cleanup counts: %+v", counts)
	}
	var oldVote, recentVote, keptIdentity, oldOrphan, featureCount, recentRequest int
	for _, check := range []struct {
		query string
		value *int
	}{
		{`SELECT COUNT(*) FROM votes WHERE identity_id='old-with-suggestion'`, &oldVote},
		{`SELECT COUNT(*) FROM votes WHERE identity_id='recent-voter'`, &recentVote},
		{`SELECT COUNT(*) FROM identities WHERE id='old-with-suggestion'`, &keptIdentity},
		{`SELECT COUNT(*) FROM identities WHERE id='old-orphan'`, &oldOrphan},
		{`SELECT COUNT(*) FROM features WHERE id='feature-family-sharing' AND status='voting'`, &featureCount},
		{`SELECT COUNT(*) FROM otp_requests WHERE email='recent@example.com'`, &recentRequest},
	} {
		if err := store.db.QueryRowContext(ctx, check.query).Scan(check.value); err != nil {
			t.Fatal(err)
		}
	}
	if oldVote != 0 || recentVote != 1 || keptIdentity != 1 || oldOrphan != 0 || featureCount != 1 || recentRequest != 1 {
		t.Fatalf("unexpected state after cleanup: old vote=%d recent vote=%d kept identity=%d orphan=%d feature=%d recent request=%d", oldVote, recentVote, keptIdentity, oldOrphan, featureCount, recentRequest)
	}
	features, err := store.ListFeatures(ctx, domain.Voting, "cat-care")
	if err != nil {
		t.Fatal(err)
	}
	for _, feature := range features {
		if feature.ID == "feature-family-sharing" && feature.VoteCount != 1 {
			t.Fatalf("public vote count after expiry = %d, want 1", feature.VoteCount)
		}
	}
	second, err := store.RunRetention(ctx, now)
	if err != nil || second != (RetentionCounts{}) {
		t.Fatalf("cleanup is not idempotent: counts=%+v err=%v", second, err)
	}
}

func TestVerificationRenewsVoteRetention(t *testing.T) {
	store := retentionTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(-1, 0, -1)
	if _, err := store.db.ExecContext(ctx, `INSERT INTO identities(id,email,created_at,last_verified_at) VALUES('renewed','renewed@example.com',$1,$1)`, old); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO votes(identity_id,feature_id,created_at) VALUES('renewed','feature-family-sharing',$1)`, old); err != nil {
		t.Fatal(err)
	}
	challenge := application.OTPChallenge{ID: "challenge", Email: "renewed@example.com", Digest: "digest", ExpiresAt: now.Add(time.Minute)}
	if allowed, err := store.SaveOTPChallenge(ctx, challenge, now); err != nil || !allowed {
		t.Fatalf("save challenge: allowed=%v err=%v", allowed, err)
	}
	if _, err := store.CompleteOTP(ctx, challenge.Email, challenge.ID, now, "unused-id", "new-session", now.Add(application.SessionValidity)); err != nil {
		t.Fatal(err)
	}
	counts, err := store.RunRetention(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Votes != 0 {
		t.Fatalf("renewed vote expired: %+v", counts)
	}
	var verified time.Time
	if err := store.db.QueryRowContext(ctx, `SELECT last_verified_at FROM identities WHERE id='renewed'`).Scan(&verified); err != nil {
		t.Fatal(err)
	}
	if !verified.Equal(now) {
		t.Fatalf("last verification = %s, want %s", verified, now)
	}
}

func TestRetentionSuggestionDeadlinesAndEmptyIdentityGrace(t *testing.T) {
	store := retentionTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	for _, identity := range []struct {
		id       string
		verified time.Time
	}{
		{"empty-expired", now.Add(-121 * 24 * time.Hour)},
		{"empty-within-grace", now.Add(-119 * 24 * time.Hour)},
		{"reviewed-expired", now.AddDate(-1, 0, -1)},
		{"pending-expired", now.AddDate(-1, 0, -1)},
		{"pending-current", now.AddDate(-1, 0, -1)},
	} {
		if _, err := store.db.ExecContext(ctx, `INSERT INTO identities(id,email,created_at,last_verified_at) VALUES($1,$2,$3,$3)`, identity.id, identity.id+"@example.com", identity.verified); err != nil {
			t.Fatal(err)
		}
	}
	for _, suggestion := range []struct {
		id, identity, status string
		created, reviewed    time.Time
	}{
		{"reviewed-old", "reviewed-expired", "rejected", now.AddDate(-2, 0, 0), now.AddDate(-1, 0, -1)},
		{"pending-old", "pending-expired", "pending_review", now.Add(-181 * 24 * time.Hour), time.Time{}},
		{"pending-new", "pending-current", "pending_review", now.Add(-179 * 24 * time.Hour), time.Time{}},
	} {
		var reviewed any
		if !suggestion.reviewed.IsZero() {
			reviewed = suggestion.reviewed
		}
		if _, err := store.db.ExecContext(ctx, `INSERT INTO suggestions(id,app_id,identity_id,title,description,status,created_at,reviewed_at) VALUES($1,'app-cat-care',$2,'idea','detail',$3,$4,$5)`, suggestion.id, suggestion.identity, suggestion.status, suggestion.created, reviewed); err != nil {
			t.Fatal(err)
		}
	}
	counts, err := store.RunRetention(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Suggestions != 2 || counts.Identities != 3 || counts.PendingReviewOverdue != 1 {
		t.Fatalf("unexpected cleanup counts: %+v", counts)
	}
	for _, check := range []struct {
		query string
		want  int
	}{
		{`SELECT COUNT(*) FROM identities WHERE id='empty-within-grace'`, 1},
		{`SELECT COUNT(*) FROM identities WHERE id='pending-current'`, 1},
		{`SELECT COUNT(*) FROM suggestions WHERE id='pending-new'`, 1},
		{`SELECT COUNT(*) FROM identities WHERE id='pending-expired'`, 0},
	} {
		var got int
		if err := store.db.QueryRowContext(ctx, check.query).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != check.want {
			t.Fatalf("%s: got %d, want %d", check.query, got, check.want)
		}
	}
}
