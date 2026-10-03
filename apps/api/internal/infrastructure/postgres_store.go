package infrastructure

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
)

//go:embed migrations/*.sql
var migrations embed.FS

type PostgresStore struct{ db *sql.DB }

func OpenPostgres(databaseURL string, seed domain.Board) (*PostgresStore, error) {
	if databaseURL == "" {
		return nil, errors.New("WISHLIST_DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	store := &PostgresStore{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err == nil {
		err = store.migrate(ctx)
	}
	if err == nil {
		err = store.seed(seed)
	}
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(971103)`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)`); err != nil {
		return err
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for index, file := range files {
		version := index + 1
		var exists bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, version).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		body, readErr := migrations.ReadFile("migrations/" + file.Name())
		if readErr != nil {
			return readErr
		}
		for _, statement := range strings.Split(string(body), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err = tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migration %s: %w", file.Name(), err)
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, version); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *PostgresStore) Close() error                   { return s.db.Close() }
func (s *PostgresStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *PostgresStore) seed(board domain.Board) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, app := range board.Apps() {
		if _, err = tx.Exec(`INSERT INTO apps VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, app.ID, app.Slug, app.Name, app.Description, app.URL, app.Active); err != nil {
			return err
		}
	}
	for _, status := range []domain.Status{domain.Voting, domain.Producing, domain.Delivered} {
		rows, e := board.Features(status, "")
		if e != nil {
			return e
		}
		for _, f := range rows {
			var delivered, link any
			if f.DeliveredAt != nil {
				delivered = f.DeliveredAt.UTC()
			}
			if f.DeliveryURL != nil {
				link = *f.DeliveryURL
			}
			var producing any
			if f.ProducingAt != nil {
				producing = f.ProducingAt.UTC()
			}
			if _, err = tx.Exec(`INSERT INTO features(id,slug,title,description,app_id,status,published_at,producing_at,delivered_at,delivery_url) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT DO NOTHING`, f.ID, f.Slug, f.Title, f.Description, f.AppID, string(f.Status), f.PublishedAt.UTC(), producing, delivered, link); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
func (s *PostgresStore) ListApps(ctx context.Context) ([]domain.App, error) {
	rows, e := s.db.QueryContext(ctx, `SELECT id,slug,name,description,url,active FROM apps WHERE active=true ORDER BY name`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.App{}
	for rows.Next() {
		var a domain.App
		if e = rows.Scan(&a.ID, &a.Slug, &a.Name, &a.Description, &a.URL, &a.Active); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *PostgresStore) ListAllApps(ctx context.Context) ([]domain.App, error) {
	rows, e := s.db.QueryContext(ctx, `SELECT id,slug,name,description,url,active FROM apps ORDER BY name`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.App{}
	for rows.Next() {
		var a domain.App
		if e = rows.Scan(&a.ID, &a.Slug, &a.Name, &a.Description, &a.URL, &a.Active); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *PostgresStore) ListFeatures(ctx context.Context, status domain.Status, slug string) ([]domain.Feature, error) {
	if !status.Valid() || (slug != "" && !domain.ValidSlug(slug)) {
		return nil, domain.ErrInvalidRequest
	}
	rows, e := s.db.QueryContext(ctx, `SELECT f.id,f.slug,f.title,f.description,f.app_id,a.slug,a.name,f.status,(SELECT COUNT(*) FROM votes v WHERE v.feature_id=f.id),f.published_at,f.producing_at,f.delivered_at,f.delivery_url FROM features f JOIN apps a ON a.id=f.app_id WHERE a.active=true AND f.status=$1 AND ($2='' OR a.slug=$3) ORDER BY (SELECT COUNT(*) FROM votes v WHERE v.feature_id=f.id) DESC,f.published_at ASC,f.id ASC`, string(status), slug, slug)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Feature{}
	for rows.Next() {
		var f domain.Feature
		var published time.Time
		var producing, delivered sql.NullTime
		var link sql.NullString
		if e = rows.Scan(&f.ID, &f.Slug, &f.Title, &f.Description, &f.AppID, &f.AppSlug, &f.AppName, &f.Status, &f.VoteCount, &published, &producing, &delivered, &link); e != nil {
			return nil, e
		}
		f.PublishedAt = published
		if producing.Valid {
			f.ProducingAt = &producing.Time
		}
		if status == domain.Delivered && delivered.Valid {
			f.DeliveredAt = &delivered.Time
		}
		if status == domain.Delivered && link.Valid {
			f.DeliveryURL = &link.String
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
func (s *PostgresStore) SaveOTPChallenge(ctx context.Context, c application.OTPChallenge, now time.Time) (bool, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return false, e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, c.Email); e != nil {
		return false, e
	}
	cut := now.Add(-time.Hour)
	var count int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM otp_requests WHERE email=$1 AND requested_at>$2`, c.Email, cut).Scan(&count); e != nil {
		return false, e
	}
	if count >= 3 {
		return false, tx.Commit()
	}
	_, e = tx.ExecContext(ctx, `DELETE FROM otp_challenges WHERE email=$1`, c.Email)
	if e != nil {
		return false, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO otp_challenges VALUES($1,$2,$3,$4,0)`, c.ID, c.Email, c.Digest, c.ExpiresAt.UTC())
	if e != nil {
		return false, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO otp_requests VALUES($1,$2)`, c.Email, now.UTC())
	if e != nil {
		return false, e
	}
	return true, tx.Commit()
}
func (s *PostgresStore) OTPChallenge(ctx context.Context, email string) (application.OTPChallenge, bool, error) {
	var c application.OTPChallenge
	var expires time.Time
	e := s.db.QueryRowContext(ctx, `SELECT id,email,digest,expires_at,attempts FROM otp_challenges WHERE email=$1`, email).Scan(&c.ID, &c.Email, &c.Digest, &expires, &c.Attempts)
	if errors.Is(e, sql.ErrNoRows) {
		return c, false, nil
	}
	if e != nil {
		return c, false, e
	}
	c.ExpiresAt = expires
	return c, true, nil
}
func (s *PostgresStore) RecordOTPFailure(ctx context.Context, email string) error {
	_, e := s.db.ExecContext(ctx, `UPDATE otp_challenges SET attempts=attempts+1 WHERE email=$1`, email)
	return e
}
func (s *PostgresStore) CompleteOTP(ctx context.Context, email, challengeID string, now time.Time, identityID, sessionID string, expires time.Time) (string, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	var id string
	var ex time.Time
	var attempts int
	e = tx.QueryRowContext(ctx, `SELECT id,expires_at,attempts FROM otp_challenges WHERE email=$1 FOR UPDATE`, email).Scan(&id, &ex, &attempts)
	validUntil := ex
	if e != nil || id != challengeID || !now.Before(validUntil) || attempts >= 5 {
		return "", domain.ErrInvalidCode
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO identities(id,email,created_at,last_verified_at) VALUES($1,$2,$3,$3) ON CONFLICT (email) DO UPDATE SET last_verified_at=$3`, identityID, email, now.UTC()); e != nil {
		return "", e
	}
	if e = tx.QueryRowContext(ctx, `SELECT id FROM identities WHERE email=$1`, email).Scan(&identityID); e != nil {
		return "", e
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO sessions VALUES($1,$2,$3)`, sessionID, identityID, expires.UTC()); e != nil {
		return "", e
	}
	_, e = tx.ExecContext(ctx, `DELETE FROM otp_challenges WHERE email=$1`, email)
	if e != nil {
		return "", e
	}
	return identityID, tx.Commit()
}
func (s *PostgresStore) VisitorSession(ctx context.Context, id string, now time.Time) (string, bool, error) {
	var identity string
	var exp time.Time
	e := s.db.QueryRowContext(ctx, `SELECT identity_id,expires_at FROM sessions WHERE id=$1`, id).Scan(&identity, &exp)
	if errors.Is(e, sql.ErrNoRows) {
		return "", false, nil
	}
	if e != nil {
		return "", false, e
	}
	until := exp
	return identity, now.Before(until), nil
}
func (s *PostgresStore) DeleteVisitorSession(ctx context.Context, id string) error {
	_, e := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id=$1`, id)
	return e
}
func (s *PostgresStore) VoteFeatureIDs(ctx context.Context, id string) ([]string, error) {
	r, e := s.db.QueryContext(ctx, `SELECT feature_id FROM votes WHERE identity_id=$1 ORDER BY feature_id`, id)
	if e != nil {
		return nil, e
	}
	defer r.Close()
	out := []string{}
	for r.Next() {
		var v string
		if e = r.Scan(&v); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, r.Err()
}
func (s *PostgresStore) ToggleVote(ctx context.Context, identity, feature string, now time.Time) (bool, int, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return false, 0, e
	}
	defer tx.Rollback()
	// Retention cleanup locks identities first, so voting takes the same lock order.
	var lockedID string
	if e = tx.QueryRowContext(ctx, `SELECT id FROM identities WHERE id=$1 FOR UPDATE`, identity).Scan(&lockedID); e != nil {
		return false, 0, e
	}
	var status string
	e = tx.QueryRowContext(ctx, `SELECT status FROM features WHERE id=$1 FOR UPDATE`, feature).Scan(&status)
	if errors.Is(e, sql.ErrNoRows) {
		return false, 0, domain.ErrFeatureNotFound
	}
	if e != nil {
		return false, 0, e
	}
	var exists int
	e = tx.QueryRowContext(ctx, `SELECT 1 FROM votes WHERE identity_id=$1 AND feature_id=$2`, identity, feature).Scan(&exists)
	voted := e == nil
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return false, 0, e
	}
	if voted {
		_, e = tx.ExecContext(ctx, `DELETE FROM votes WHERE identity_id=$1 AND feature_id=$2`, identity, feature)
	} else {
		if status != string(domain.Voting) {
			return false, 0, domain.ErrFeatureNotVoting
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO votes VALUES($1,$2,$3)`, identity, feature, now.UTC())
		voted = true
	}
	if e != nil {
		return false, 0, e
	}
	var count int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM votes WHERE feature_id=$1`, feature).Scan(&count); e != nil {
		return false, 0, e
	}
	return voted, count, tx.Commit()
}
func (s *PostgresStore) CreateSuggestion(ctx context.Context, id, app, identity, title, description string, now time.Time) (string, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	var lockedID string
	if e = tx.QueryRowContext(ctx, `SELECT id FROM identities WHERE id=$1 FOR UPDATE`, identity).Scan(&lockedID); e != nil {
		return "", e
	}
	if _, e = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 1))`, identity); e != nil {
		return "", e
	}
	var active bool
	e = tx.QueryRowContext(ctx, `SELECT active FROM apps WHERE id=$1`, app).Scan(&active)
	if errors.Is(e, sql.ErrNoRows) {
		return "", domain.ErrAppNotFound
	}
	if e != nil {
		return "", e
	}
	if !active {
		return "", domain.ErrAppNotFound
	}
	var n int
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM suggestions WHERE identity_id=$1 AND created_at>$2`, identity, now.Add(-time.Hour).UTC()).Scan(&n)
	if e != nil {
		return "", e
	}
	if n >= 3 {
		return "", domain.ErrRateLimited
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO suggestions(id,app_id,identity_id,title,description,created_at) VALUES($1,$2,$3,$4,$5,$6)`, id, app, identity, title, description, now.UTC())
	if e != nil {
		return "", fmt.Errorf("insert suggestion: %w", e)
	}
	return id, tx.Commit()
}

func (s *PostgresStore) CreateApp(ctx context.Context, a domain.App) error {
	_, e := s.db.ExecContext(ctx, `INSERT INTO apps(id,slug,name,description,url,active) VALUES($1,$2,$3,$4,$5,true)`, a.ID, a.Slug, a.Name, a.Description, a.URL)
	return e
}
func (s *PostgresStore) UpdateApp(ctx context.Context, id, name, description, url string, active bool) error {
	r, e := s.db.ExecContext(ctx, `UPDATE apps SET name=$1,description=$2,url=$3,active=$4 WHERE id=$5`, name, description, url, active, id)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e == nil && n == 0 {
		return domain.ErrAppNotFound
	}
	return e
}
func (s *PostgresStore) CreateFeature(ctx context.Context, f domain.Feature) error {
	_, e := s.db.ExecContext(ctx, `INSERT INTO features(id,slug,title,description,app_id,status,published_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, f.ID, f.Slug, f.Title, f.Description, f.AppID, string(f.Status), f.PublishedAt.UTC())
	return e
}
func (s *PostgresStore) UpdateFeature(ctx context.Context, id, title, description string, status domain.Status, deliveryURL string, now time.Time) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var current string
	e = tx.QueryRowContext(ctx, `SELECT status FROM features WHERE id=$1 FOR UPDATE`, id).Scan(&current)
	if errors.Is(e, sql.ErrNoRows) {
		return domain.ErrFeatureNotFound
	}
	if e != nil {
		return e
	}
	order := map[string]int{string(domain.Voting): 0, string(domain.Producing): 1, string(domain.Delivered): 2}
	if order[string(status)] != order[current] && order[string(status)] != order[current]+1 {
		return domain.ErrInvalidTransition
	}
	var delivered any
	var producing any
	if status == domain.Producing && current != string(domain.Producing) {
		producing = now.UTC()
	}
	if status == domain.Delivered && current != string(domain.Delivered) {
		delivered = now.UTC()
	}
	_, e = tx.ExecContext(ctx, `UPDATE features SET title=$1,description=$2,status=$3,producing_at=COALESCE($4::timestamptz, producing_at),delivered_at=COALESCE($5::timestamptz, delivered_at),delivery_url=CASE WHEN $6='' THEN delivery_url ELSE $6 END WHERE id=$7`, title, description, string(status), producing, delivered, deliveryURL, id)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (s *PostgresStore) PendingSuggestions(ctx context.Context) ([]map[string]any, error) {
	rows, e := s.db.QueryContext(ctx, `SELECT id,app_id,title,description,created_at FROM suggestions WHERE status='pending_review' ORDER BY created_at`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, app, title, description string
		var created time.Time
		if e = rows.Scan(&id, &app, &title, &description, &created); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": id, "app_id": app, "title": title, "description": description, "created_at": created})
	}
	return out, rows.Err()
}
func (s *PostgresStore) ReviewSuggestion(ctx context.Context, id, action, featureID, slug, title, description string, now time.Time) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var app, status string
	e = tx.QueryRowContext(ctx, `SELECT app_id,status FROM suggestions WHERE id=$1 FOR UPDATE`, id).Scan(&app, &status)
	if errors.Is(e, sql.ErrNoRows) {
		return domain.ErrInvalidRequest
	}
	if e != nil {
		return e
	}
	if status != "pending_review" {
		return domain.ErrInvalidRequest
	}
	switch action {
	case "reject":
		_, e = tx.ExecContext(ctx, `UPDATE suggestions SET status='rejected',reviewed_at=$1 WHERE id=$2`, now.UTC(), id)
	case "accept":
		_, e = tx.ExecContext(ctx, `INSERT INTO features(id,slug,title,description,app_id,status,published_at) VALUES($1,$2,$3,$4,$5,'voting',$6)`, featureID, slug, title, description, app, now.UTC())
		if e == nil {
			_, e = tx.ExecContext(ctx, `UPDATE suggestions SET status='accepted',resulting_feature_id=$1,reviewed_at=$2 WHERE id=$3`, featureID, now.UTC(), id)
		}
	case "merge":
		var featureApp string
		e = tx.QueryRowContext(ctx, `SELECT app_id FROM features WHERE id=$1`, featureID).Scan(&featureApp)
		if errors.Is(e, sql.ErrNoRows) || featureApp != app {
			return domain.ErrInvalidRequest
		}
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `UPDATE suggestions SET status='merged',resulting_feature_id=$1,reviewed_at=$2 WHERE id=$3`, featureID, now.UTC(), id)
	default:
		return domain.ErrInvalidRequest
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}
