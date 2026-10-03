package infrastructure

import (
	"context"
	"database/sql"
	"time"
)

// RetentionCounts contains aggregate deletion counts only; it is safe to log.
type RetentionCounts struct {
	OTPChallenges        int64
	OTPRequests          int64
	Sessions             int64
	Suggestions          int64
	PendingReviewOverdue int64
	Votes                int64
	Identities           int64
}

// RunRetention removes expired visitor records in one transaction. It does not
// change feature lifecycle, and public vote totals follow the votes table.
func (s *PostgresStore) RunRetention(ctx context.Context, now time.Time) (RetentionCounts, error) {
	var counts RetentionCounts
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return counts, err
	}
	defer tx.Rollback()
	var acquired bool
	if err = tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock(971104)`).Scan(&acquired); err != nil {
		return counts, err
	}
	if !acquired {
		return counts, nil
	}
	now = now.UTC()
	// Remove technical records as soon as their useful period ends. A periodic
	// worker gives the 24-hour maximum in normal operation without keeping them
	// for an extra 24 hours by design.
	for _, deletion := range []struct {
		query string
		cut   time.Time
		count *int64
	}{
		{`DELETE FROM otp_challenges WHERE expires_at <= $1`, now, &counts.OTPChallenges},
		// The throttle needs one hour. Keep a 23-hour cutoff so an hourly run
		// removes rows within the approved 24-hour maximum in normal operation.
		{`DELETE FROM otp_requests WHERE requested_at <= $1`, now.Add(-23 * time.Hour), &counts.OTPRequests},
		{`DELETE FROM sessions WHERE expires_at <= $1`, now, &counts.Sessions},
		{`DELETE FROM suggestions WHERE status='pending_review' AND created_at <= $1`, now.Add(-180 * 24 * time.Hour), &counts.Suggestions},
		{`DELETE FROM suggestions WHERE status <> 'pending_review' AND reviewed_at <= $1`, now.AddDate(-1, 0, 0), &counts.Suggestions},
	} {
		var affected int64
		affected, err = deleteBefore(ctx, tx, deletion.query, deletion.cut)
		if err != nil {
			return RetentionCounts{}, err
		}
		*deletion.count += affected
	}
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM suggestions WHERE status='pending_review' AND created_at <= $1`, now.Add(-90*24*time.Hour)).Scan(&counts.PendingReviewOverdue); err != nil {
		return RetentionCounts{}, err
	}
	// Verification, voting, and suggestion submission also lock the identity.
	// Hold these locks until commit so a newly renewed identity keeps its votes.
	orphanCutoff := now.Add(-120 * 24 * time.Hour) // 30-day session + 90 days
	rows, err := tx.QueryContext(ctx, `SELECT id FROM identities WHERE last_verified_at <= $1 FOR UPDATE`, orphanCutoff)
	if err != nil {
		return RetentionCounts{}, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return RetentionCounts{}, err
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return RetentionCounts{}, err
	}
	counts.Votes, err = deleteBefore(ctx, tx, `DELETE FROM votes v USING identities i WHERE v.identity_id=i.id AND i.last_verified_at <= $1`, now.AddDate(-1, 0, 0))
	if err != nil {
		return RetentionCounts{}, err
	}
	counts.Identities, err = deleteBefore(ctx, tx, `DELETE FROM identities i WHERE i.last_verified_at <= $1 AND NOT EXISTS (SELECT 1 FROM votes v WHERE v.identity_id=i.id) AND NOT EXISTS (SELECT 1 FROM suggestions s WHERE s.identity_id=i.id) AND NOT EXISTS (SELECT 1 FROM sessions x WHERE x.identity_id=i.id)`, orphanCutoff)
	if err != nil {
		return RetentionCounts{}, err
	}
	if err = tx.Commit(); err != nil {
		return RetentionCounts{}, err
	}
	return counts, nil
}

func deleteBefore(ctx context.Context, tx *sql.Tx, query string, cutoff time.Time) (int64, error) {
	result, err := tx.ExecContext(ctx, query, cutoff.UTC())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
