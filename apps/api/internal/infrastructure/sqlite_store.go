package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct{ db *sql.DB }

func OpenSQLite(path string, seed domain.Board) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &SQLiteStore{db: db}
	if _, err = db.Exec(`PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;
CREATE TABLE IF NOT EXISTS apps(id TEXT PRIMARY KEY,slug TEXT UNIQUE NOT NULL,name TEXT NOT NULL,description TEXT NOT NULL,url TEXT NOT NULL,active INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS features(id TEXT PRIMARY KEY,slug TEXT NOT NULL,title TEXT NOT NULL,description TEXT NOT NULL,app_id TEXT NOT NULL REFERENCES apps(id),status TEXT NOT NULL,published_at TEXT NOT NULL,producing_at TEXT,delivered_at TEXT,delivery_url TEXT);
CREATE TABLE IF NOT EXISTS identities(id TEXT PRIMARY KEY,email TEXT UNIQUE NOT NULL,created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS otp_challenges(id TEXT PRIMARY KEY,email TEXT NOT NULL UNIQUE,digest TEXT NOT NULL,expires_at TEXT NOT NULL,attempts INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS otp_requests(email TEXT NOT NULL,requested_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS sessions(id TEXT PRIMARY KEY,identity_id TEXT NOT NULL REFERENCES identities(id),expires_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS votes(identity_id TEXT NOT NULL REFERENCES identities(id),feature_id TEXT NOT NULL REFERENCES features(id),created_at TEXT NOT NULL,PRIMARY KEY(identity_id,feature_id));
CREATE TABLE IF NOT EXISTS suggestions(id TEXT PRIMARY KEY,app_id TEXT NOT NULL REFERENCES apps(id),identity_id TEXT NOT NULL REFERENCES identities(id),title TEXT NOT NULL,description TEXT NOT NULL,status TEXT NOT NULL DEFAULT 'pending_review',resulting_feature_id TEXT,created_at TEXT NOT NULL,reviewed_at TEXT);`); err != nil {
		db.Close()
		return nil, err
	}
	for _, column := range []struct{ table, name, definition string }{{"suggestions", "resulting_feature_id", "TEXT"}, {"suggestions", "reviewed_at", "TEXT"}, {"features", "producing_at", "TEXT"}} {
		if err := ensureColumn(db, column.table, column.name, column.definition); err != nil {
			db.Close()
			return nil, err
		}
	}
	if _, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS ux_features_app_slug ON features(app_id,slug)`); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.seed(seed); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}
func ensureColumn(db *sql.DB, table, name, definition string) error {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var cid, notnull, pk int
		var column, kind string
		var dflt any
		if err = rows.Scan(&cid, &column, &kind, &notnull, &dflt, &pk); err != nil {
			rows.Close()
			return err
		}
		if column == name {
			found = true
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if !found {
		_, err = db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + name + ` ` + definition)
	}
	return err
}
func (s *SQLiteStore) Close() error                   { return s.db.Close() }
func (s *SQLiteStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *SQLiteStore) seed(board domain.Board) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, app := range board.Apps() {
		if _, err = tx.Exec(`INSERT OR IGNORE INTO apps VALUES(?,?,?,?,?,?)`, app.ID, app.Slug, app.Name, app.Description, app.URL, app.Active); err != nil {
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
				delivered = f.DeliveredAt.UTC().Format(time.RFC3339Nano)
			}
			if f.DeliveryURL != nil {
				link = *f.DeliveryURL
			}
			var producing any
			if f.ProducingAt != nil {
				producing = f.ProducingAt.UTC().Format(time.RFC3339Nano)
			}
			if _, err = tx.Exec(`INSERT OR IGNORE INTO features(id,slug,title,description,app_id,status,published_at,producing_at,delivered_at,delivery_url) VALUES(?,?,?,?,?,?,?,?,?,?)`, f.ID, f.Slug, f.Title, f.Description, f.AppID, string(f.Status), f.PublishedAt.UTC().Format(time.RFC3339Nano), producing, delivered, link); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
func (s *SQLiteStore) ListApps(ctx context.Context) ([]domain.App, error) {
	rows, e := s.db.QueryContext(ctx, `SELECT id,slug,name,description,url,active FROM apps WHERE active=1 ORDER BY name`)
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
func (s *SQLiteStore) ListAllApps(ctx context.Context) ([]domain.App, error) {
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
func (s *SQLiteStore) ListFeatures(ctx context.Context, status domain.Status, slug string) ([]domain.Feature, error) {
	if !status.Valid() || (slug != "" && !domain.ValidSlug(slug)) {
		return nil, domain.ErrInvalidRequest
	}
	rows, e := s.db.QueryContext(ctx, `SELECT f.id,f.slug,f.title,f.description,f.app_id,a.slug,a.name,f.status,(SELECT COUNT(*) FROM votes v WHERE v.feature_id=f.id),f.published_at,f.producing_at,f.delivered_at,f.delivery_url FROM features f JOIN apps a ON a.id=f.app_id WHERE a.active=1 AND f.status=? AND (?='' OR a.slug=?) ORDER BY (SELECT COUNT(*) FROM votes v WHERE v.feature_id=f.id) DESC,f.published_at ASC,f.id ASC`, string(status), slug, slug)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Feature{}
	for rows.Next() {
		var f domain.Feature
		var published string
		var producing, delivered, link sql.NullString
		if e = rows.Scan(&f.ID, &f.Slug, &f.Title, &f.Description, &f.AppID, &f.AppSlug, &f.AppName, &f.Status, &f.VoteCount, &published, &producing, &delivered, &link); e != nil {
			return nil, e
		}
		f.PublishedAt, _ = time.Parse(time.RFC3339Nano, published)
		if producing.Valid {
			v, _ := time.Parse(time.RFC3339Nano, producing.String)
			f.ProducingAt = &v
		}
		if status == domain.Delivered && delivered.Valid {
			v, _ := time.Parse(time.RFC3339Nano, delivered.String)
			f.DeliveredAt = &v
		}
		if status == domain.Delivered && link.Valid {
			f.DeliveryURL = &link.String
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
func (s *SQLiteStore) SaveOTPChallenge(ctx context.Context, c application.OTPChallenge, now time.Time) (bool, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return false, e
	}
	defer tx.Rollback()
	cut := now.Add(-time.Hour).Format(time.RFC3339Nano)
	var count int
	if e = tx.QueryRow(`SELECT COUNT(*) FROM otp_requests WHERE email=? AND requested_at>?`, c.Email, cut).Scan(&count); e != nil {
		return false, e
	}
	if count >= 3 {
		return false, tx.Commit()
	}
	_, e = tx.Exec(`DELETE FROM otp_challenges WHERE email=?`, c.Email)
	if e != nil {
		return false, e
	}
	_, e = tx.Exec(`INSERT INTO otp_challenges VALUES(?,?,?,?,0)`, c.ID, c.Email, c.Digest, c.ExpiresAt.UTC().Format(time.RFC3339Nano))
	if e != nil {
		return false, e
	}
	_, e = tx.Exec(`INSERT INTO otp_requests VALUES(?,?)`, c.Email, now.UTC().Format(time.RFC3339Nano))
	if e != nil {
		return false, e
	}
	return true, tx.Commit()
}
func (s *SQLiteStore) OTPChallenge(ctx context.Context, email string) (application.OTPChallenge, bool, error) {
	var c application.OTPChallenge
	var expires string
	e := s.db.QueryRowContext(ctx, `SELECT id,email,digest,expires_at,attempts FROM otp_challenges WHERE email=?`, email).Scan(&c.ID, &c.Email, &c.Digest, &expires, &c.Attempts)
	if errors.Is(e, sql.ErrNoRows) {
		return c, false, nil
	}
	if e != nil {
		return c, false, e
	}
	c.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	return c, true, nil
}
func (s *SQLiteStore) RecordOTPFailure(ctx context.Context, email string) error {
	_, e := s.db.ExecContext(ctx, `UPDATE otp_challenges SET attempts=attempts+1 WHERE email=?`, email)
	return e
}
func (s *SQLiteStore) CompleteOTP(ctx context.Context, email, challengeID string, now time.Time, identityID, sessionID string, expires time.Time) (string, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	var id string
	var ex string
	var attempts int
	e = tx.QueryRowContext(ctx, `SELECT id,expires_at,attempts FROM otp_challenges WHERE email=?`, email).Scan(&id, &ex, &attempts)
	validUntil, _ := time.Parse(time.RFC3339Nano, ex)
	if e != nil || id != challengeID || !now.Before(validUntil) || attempts >= 5 {
		return "", domain.ErrInvalidCode
	}
	if _, e = tx.ExecContext(ctx, `INSERT OR IGNORE INTO identities VALUES(?,?,?)`, identityID, email, now.UTC().Format(time.RFC3339Nano)); e != nil {
		return "", e
	}
	if e = tx.QueryRowContext(ctx, `SELECT id FROM identities WHERE email=?`, email).Scan(&identityID); e != nil {
		return "", e
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO sessions VALUES(?,?,?)`, sessionID, identityID, expires.UTC().Format(time.RFC3339Nano)); e != nil {
		return "", e
	}
	_, e = tx.ExecContext(ctx, `DELETE FROM otp_challenges WHERE email=?`, email)
	if e != nil {
		return "", e
	}
	return identityID, tx.Commit()
}
func (s *SQLiteStore) VisitorSession(ctx context.Context, id string, now time.Time) (string, bool, error) {
	var identity, exp string
	e := s.db.QueryRowContext(ctx, `SELECT identity_id,expires_at FROM sessions WHERE id=?`, id).Scan(&identity, &exp)
	if errors.Is(e, sql.ErrNoRows) {
		return "", false, nil
	}
	if e != nil {
		return "", false, e
	}
	until, _ := time.Parse(time.RFC3339Nano, exp)
	return identity, now.Before(until), nil
}
func (s *SQLiteStore) DeleteVisitorSession(ctx context.Context, id string) error {
	_, e := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id=?`, id)
	return e
}
func (s *SQLiteStore) VoteFeatureIDs(ctx context.Context, id string) ([]string, error) {
	r, e := s.db.QueryContext(ctx, `SELECT feature_id FROM votes WHERE identity_id=? ORDER BY feature_id`, id)
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
func (s *SQLiteStore) ToggleVote(ctx context.Context, identity, feature string, now time.Time) (bool, int, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return false, 0, e
	}
	defer tx.Rollback()
	var status string
	e = tx.QueryRowContext(ctx, `SELECT status FROM features WHERE id=?`, feature).Scan(&status)
	if errors.Is(e, sql.ErrNoRows) {
		return false, 0, domain.ErrFeatureNotFound
	}
	if e != nil {
		return false, 0, e
	}
	var exists int
	e = tx.QueryRowContext(ctx, `SELECT 1 FROM votes WHERE identity_id=? AND feature_id=?`, identity, feature).Scan(&exists)
	voted := e == nil
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return false, 0, e
	}
	if voted {
		_, e = tx.ExecContext(ctx, `DELETE FROM votes WHERE identity_id=? AND feature_id=?`, identity, feature)
	} else {
		if status != string(domain.Voting) {
			return false, 0, domain.ErrFeatureNotVoting
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO votes VALUES(?,?,?)`, identity, feature, now.UTC().Format(time.RFC3339Nano))
		voted = true
	}
	if e != nil {
		return false, 0, e
	}
	var count int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM votes WHERE feature_id=?`, feature).Scan(&count); e != nil {
		return false, 0, e
	}
	return voted, count, tx.Commit()
}
func (s *SQLiteStore) CreateSuggestion(ctx context.Context, id, app, identity, title, description string, now time.Time) (string, error) {
	var active int
	e := s.db.QueryRowContext(ctx, `SELECT active FROM apps WHERE id=?`, app).Scan(&active)
	if errors.Is(e, sql.ErrNoRows) || active != 1 {
		return "", domain.ErrAppNotFound
	}
	if e != nil {
		return "", e
	}
	var n int
	e = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM suggestions WHERE identity_id=? AND created_at>?`, identity, now.Add(-time.Hour).UTC().Format(time.RFC3339Nano)).Scan(&n)
	if e != nil {
		return "", e
	}
	if n >= 3 {
		return "", domain.ErrRateLimited
	}
	_, e = s.db.ExecContext(ctx, `INSERT INTO suggestions(id,app_id,identity_id,title,description,created_at) VALUES(?,?,?,?,?,?)`, id, app, identity, title, description, now.UTC().Format(time.RFC3339Nano))
	if e != nil {
		return "", fmt.Errorf("insert suggestion: %w", e)
	}
	return id, nil
}

func (s *SQLiteStore) CreateApp(ctx context.Context, a domain.App) error {
	_, e := s.db.ExecContext(ctx, `INSERT INTO apps(id,slug,name,description,url,active) VALUES(?,?,?,?,?,1)`, a.ID, a.Slug, a.Name, a.Description, a.URL)
	return e
}
func (s *SQLiteStore) UpdateApp(ctx context.Context, id, name, description, url string, active bool) error {
	r, e := s.db.ExecContext(ctx, `UPDATE apps SET name=?,description=?,url=?,active=? WHERE id=?`, name, description, url, active, id)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e == nil && n == 0 {
		return domain.ErrAppNotFound
	}
	return e
}
func (s *SQLiteStore) CreateFeature(ctx context.Context, f domain.Feature) error {
	_, e := s.db.ExecContext(ctx, `INSERT INTO features(id,slug,title,description,app_id,status,published_at) VALUES(?,?,?,?,?,?,?)`, f.ID, f.Slug, f.Title, f.Description, f.AppID, string(f.Status), f.PublishedAt.UTC().Format(time.RFC3339Nano))
	return e
}
func (s *SQLiteStore) UpdateFeature(ctx context.Context, id, title, description string, status domain.Status, deliveryURL string, now time.Time) error {
	var current string
	e := s.db.QueryRowContext(ctx, `SELECT status FROM features WHERE id=?`, id).Scan(&current)
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
		producing = now.UTC().Format(time.RFC3339Nano)
	}
	if status == domain.Delivered && current != string(domain.Delivered) {
		delivered = now.UTC().Format(time.RFC3339Nano)
	}
	_, e = s.db.ExecContext(ctx, `UPDATE features SET title=?,description=?,status=?,producing_at=CASE WHEN ? IS NULL THEN producing_at ELSE ? END,delivered_at=CASE WHEN ? IS NULL THEN delivered_at ELSE ? END,delivery_url=CASE WHEN ?='' THEN delivery_url ELSE ? END WHERE id=?`, title, description, string(status), producing, producing, delivered, delivered, deliveryURL, deliveryURL, id)
	return e
}
func (s *SQLiteStore) PendingSuggestions(ctx context.Context) ([]map[string]any, error) {
	rows, e := s.db.QueryContext(ctx, `SELECT id,app_id,title,description,created_at FROM suggestions WHERE status='pending_review' ORDER BY created_at`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, app, title, description, created string
		if e = rows.Scan(&id, &app, &title, &description, &created); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": id, "app_id": app, "title": title, "description": description, "created_at": created})
	}
	return out, rows.Err()
}
func (s *SQLiteStore) ReviewSuggestion(ctx context.Context, id, action, featureID, slug, title, description string, now time.Time) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var app, status string
	e = tx.QueryRowContext(ctx, `SELECT app_id,status FROM suggestions WHERE id=?`, id).Scan(&app, &status)
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
		_, e = tx.ExecContext(ctx, `UPDATE suggestions SET status='rejected',reviewed_at=? WHERE id=?`, now.UTC().Format(time.RFC3339Nano), id)
	case "accept":
		_, e = tx.ExecContext(ctx, `INSERT INTO features(id,slug,title,description,app_id,status,published_at) VALUES(?,?,?,?,?,'voting',?)`, featureID, slug, title, description, app, now.UTC().Format(time.RFC3339Nano))
		if e == nil {
			_, e = tx.ExecContext(ctx, `UPDATE suggestions SET status='accepted',resulting_feature_id=?,reviewed_at=? WHERE id=?`, featureID, now.UTC().Format(time.RFC3339Nano), id)
		}
	case "merge":
		var featureApp string
		e = tx.QueryRowContext(ctx, `SELECT app_id FROM features WHERE id=?`, featureID).Scan(&featureApp)
		if errors.Is(e, sql.ErrNoRows) || featureApp != app {
			return domain.ErrInvalidRequest
		}
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `UPDATE suggestions SET status='merged',resulting_feature_id=?,reviewed_at=? WHERE id=?`, featureID, now.UTC().Format(time.RFC3339Nano), id)
	default:
		return domain.ErrInvalidRequest
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}
