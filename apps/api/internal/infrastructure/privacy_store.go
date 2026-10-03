package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
)

type privacyQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type privacyIdentity struct {
	id             string
	email          string
	createdAt      time.Time
	lastVerifiedAt time.Time
}

func currentPrivacyIdentity(ctx context.Context, q privacyQuerier, sessionID string, now time.Time, lock bool) (privacyIdentity, error) {
	query := `SELECT i.id,i.email,i.created_at,i.last_verified_at,s.expires_at
		FROM sessions s JOIN identities i ON i.id=s.identity_id WHERE s.id=$1`
	if lock {
		query += ` FOR UPDATE OF i`
	}
	var identity privacyIdentity
	var expires time.Time
	err := q.QueryRowContext(ctx, query, sessionID).Scan(&identity.id, &identity.email, &identity.createdAt, &identity.lastVerifiedAt, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return privacyIdentity{}, domain.ErrUnauthenticated
	}
	if err != nil {
		return privacyIdentity{}, err
	}
	if !now.Before(expires) {
		return privacyIdentity{}, domain.ErrUnauthenticated
	}
	verifiedAt := expires.Add(-application.SessionValidity)
	if now.Before(verifiedAt) || !now.Before(verifiedAt.Add(application.PrivacyFreshness)) {
		return privacyIdentity{}, domain.ErrFreshVerification
	}
	return identity, nil
}

func (s *PostgresStore) PrivacySession(ctx context.Context, sessionID string, now time.Time) (string, error) {
	identity, err := currentPrivacyIdentity(ctx, s.db, sessionID, now.UTC(), false)
	return identity.email, err
}

func (s *PostgresStore) PrivacyData(ctx context.Context, sessionID string, now time.Time) (application.PrivacyData, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return application.PrivacyData{}, err
	}
	defer tx.Rollback()
	identity, err := currentPrivacyIdentity(ctx, tx, sessionID, now.UTC(), false)
	if err != nil {
		return application.PrivacyData{}, err
	}
	data, err := privacyDataForIdentity(ctx, tx, identity, now)
	if err != nil {
		return application.PrivacyData{}, err
	}
	return data, tx.Commit()
}

func (s *PostgresStore) OperatorPrivacyData(ctx context.Context, email string, now time.Time) (application.PrivacyData, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return application.PrivacyData{}, err
	}
	defer tx.Rollback()
	var identity privacyIdentity
	err = tx.QueryRowContext(ctx, `SELECT id,email,created_at,last_verified_at FROM identities WHERE email=$1`, email).Scan(&identity.id, &identity.email, &identity.createdAt, &identity.lastVerifiedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return application.PrivacyData{}, domain.ErrInvalidRequest
	}
	if err != nil {
		return application.PrivacyData{}, err
	}
	data, err := privacyDataForIdentity(ctx, tx, identity, now)
	if err != nil {
		return application.PrivacyData{}, err
	}
	return data, tx.Commit()
}

func privacyDataForIdentity(ctx context.Context, tx *sql.Tx, identity privacyIdentity, now time.Time) (application.PrivacyData, error) {
	data := application.PrivacyData{Votes: []application.PrivacyVote{}, Suggestions: []application.PrivacySuggestion{}}
	data.Email, data.CreatedAt, data.LastVerifiedAt = identity.email, identity.createdAt, identity.lastVerifiedAt
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE identity_id=$1 AND expires_at>$2`, identity.id, now.UTC()).Scan(&data.ActiveSessions); err != nil {
		return application.PrivacyData{}, err
	}
	votes, err := tx.QueryContext(ctx, `SELECT f.id,a.name,f.title,f.status,v.created_at
		FROM votes v JOIN features f ON f.id=v.feature_id JOIN apps a ON a.id=f.app_id
		WHERE v.identity_id=$1 ORDER BY v.created_at,f.id`, identity.id)
	if err != nil {
		return application.PrivacyData{}, err
	}
	for votes.Next() {
		var vote application.PrivacyVote
		if err = votes.Scan(&vote.FeatureID, &vote.AppName, &vote.Title, &vote.Status, &vote.CreatedAt); err != nil {
			votes.Close()
			return application.PrivacyData{}, err
		}
		data.Votes = append(data.Votes, vote)
	}
	err = votes.Err()
	votes.Close()
	if err != nil {
		return application.PrivacyData{}, err
	}
	suggestions, err := tx.QueryContext(ctx, `SELECT s.id,a.name,s.title,s.description,s.status,s.created_at,s.reviewed_at,s.resulting_feature_id
		FROM suggestions s JOIN apps a ON a.id=s.app_id
		WHERE s.identity_id=$1 ORDER BY s.created_at,s.id`, identity.id)
	if err != nil {
		return application.PrivacyData{}, err
	}
	for suggestions.Next() {
		var suggestion application.PrivacySuggestion
		var reviewed sql.NullTime
		var featureID sql.NullString
		if err = suggestions.Scan(&suggestion.ID, &suggestion.AppName, &suggestion.Title, &suggestion.Description, &suggestion.Status, &suggestion.CreatedAt, &reviewed, &featureID); err != nil {
			suggestions.Close()
			return application.PrivacyData{}, err
		}
		if reviewed.Valid {
			suggestion.ReviewedAt = &reviewed.Time
		}
		if featureID.Valid {
			suggestion.ResultingFeatureID = &featureID.String
		}
		data.Suggestions = append(data.Suggestions, suggestion)
	}
	err = suggestions.Err()
	suggestions.Close()
	if err != nil {
		return application.PrivacyData{}, err
	}
	return data, nil
}

func (s *PostgresStore) CompleteEmailCorrection(ctx context.Context, sessionID, newEmail, challengeID string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id string
	var expires time.Time
	var attempts int
	err = tx.QueryRowContext(ctx, `SELECT id,expires_at,attempts FROM otp_challenges WHERE email=$1 FOR UPDATE`, newEmail).Scan(&id, &expires, &attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrInvalidCode
	}
	if err != nil {
		return err
	}
	if id != challengeID || !now.Before(expires) || attempts >= 5 {
		return domain.ErrInvalidCode
	}
	identity, err := currentPrivacyIdentity(ctx, tx, sessionID, now.UTC(), true)
	if err != nil {
		return err
	}
	if identity.email == newEmail {
		return domain.ErrInvalidRequest
	}
	var inUse bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM identities WHERE email=$1)`, newEmail).Scan(&inUse); err != nil {
		return err
	}
	if inUse {
		return domain.ErrEmailInUse
	}
	_, err = tx.ExecContext(ctx, `UPDATE identities SET email=$1,last_verified_at=$2 WHERE id=$3`, newEmail, now.UTC(), identity.id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrEmailInUse
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE identity_id=$1 AND id<>$2`, identity.id, sessionID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM otp_challenges WHERE email=$1`, newEmail); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) ErasePrivacyData(ctx context.Context, sessionID string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// OTP verification locks the challenge before the identity. Keep the same
	// order so a concurrent verification cannot deadlock with erasure.
	var priorEmail string
	err = tx.QueryRowContext(ctx, `SELECT i.email FROM sessions s JOIN identities i ON i.id=s.identity_id WHERE s.id=$1`, sessionID).Scan(&priorEmail)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrUnauthenticated
	}
	if err != nil {
		return err
	}
	if err = lockPrivacyEmail(ctx, tx, priorEmail); err != nil {
		return err
	}
	identity, err := currentPrivacyIdentity(ctx, tx, sessionID, now.UTC(), true)
	if err != nil {
		return err
	}
	if identity.email != priorEmail {
		return domain.ErrInvalidRequest
	}
	if err = eraseIdentity(ctx, tx, identity, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) OperatorErasePrivacyData(ctx context.Context, email string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockPrivacyEmail(ctx, tx, email); err != nil {
		return err
	}
	var identity privacyIdentity
	err = tx.QueryRowContext(ctx, `SELECT id,email,created_at,last_verified_at FROM identities WHERE email=$1 FOR UPDATE`, email).Scan(&identity.id, &identity.email, &identity.createdAt, &identity.lastVerifiedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrInvalidRequest
	}
	if err != nil {
		return err
	}
	if err = eraseIdentity(ctx, tx, identity, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) OperatorCorrectEmail(ctx context.Context, email, newEmail string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// A concurrent OTP verification locks the challenge before the identity.
	if err = lockPrivacyEmail(ctx, tx, email); err != nil {
		return err
	}
	var identityID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM identities WHERE email=$1 FOR UPDATE`, email).Scan(&identityID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrInvalidRequest
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE identities SET email=$1 WHERE id=$2`, newEmail, identityID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrEmailInUse
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE identity_id=$1`, identityID); err != nil {
		return err
	}
	return tx.Commit()
}

func lockPrivacyEmail(ctx context.Context, tx *sql.Tx, email string) error {
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, email); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM otp_challenges WHERE email=$1`, email)
	return err
}

func eraseIdentity(ctx context.Context, tx *sql.Tx, identity privacyIdentity, now time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO privacy_public_reviews(feature_id,requested_at,status)
	SELECT DISTINCT resulting_feature_id,$2::timestamptz,'pending' FROM suggestions
		WHERE identity_id=$1 AND resulting_feature_id IS NOT NULL
		ON CONFLICT(feature_id) DO UPDATE SET requested_at=EXCLUDED.requested_at,status='pending',reviewed_at=NULL`, identity.id, now.UTC())
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM otp_requests WHERE email=$1`, identity.email); err != nil {
		return err
	}
	for _, query := range []string{
		`DELETE FROM votes WHERE identity_id=$1`,
		`DELETE FROM suggestions WHERE identity_id=$1`,
		`DELETE FROM sessions WHERE identity_id=$1`,
	} {
		if _, err = tx.ExecContext(ctx, query, identity.id); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM identities WHERE id=$1`, identity.id); err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) PendingPrivacyReviews(ctx context.Context) ([]application.PublicPrivacyReview, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT p.feature_id,a.name,f.title,f.description,f.status,COALESCE(f.delivery_url,''),p.requested_at
		FROM privacy_public_reviews p JOIN features f ON f.id=p.feature_id JOIN apps a ON a.id=f.app_id
		WHERE p.status='pending' ORDER BY p.requested_at,p.feature_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []application.PublicPrivacyReview{}
	for rows.Next() {
		var row application.PublicPrivacyReview
		if err = rows.Scan(&row.FeatureID, &row.AppName, &row.Title, &row.Description, &row.Status, &row.DeliveryURL, &row.RequestedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *PostgresStore) ResolvePrivacyReview(ctx context.Context, featureID string, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE privacy_public_reviews SET status='reviewed',reviewed_at=$2 WHERE feature_id=$1 AND status='pending'`, featureID, now.UTC())
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrInvalidRequest
	}
	return nil
}
