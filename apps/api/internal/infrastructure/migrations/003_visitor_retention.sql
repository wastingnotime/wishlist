ALTER TABLE identities ADD COLUMN last_verified_at TIMESTAMPTZ;
UPDATE identities i SET last_verified_at = GREATEST(
  i.created_at,
  COALESCE((SELECT MAX(s.expires_at - INTERVAL '30 days') FROM sessions s WHERE s.identity_id = i.id), i.created_at)
);
ALTER TABLE identities ALTER COLUMN last_verified_at SET NOT NULL;
CREATE INDEX identities_last_verified_at ON identities (last_verified_at);
CREATE INDEX otp_challenges_expires_at ON otp_challenges (expires_at);
CREATE INDEX otp_requests_requested_at ON otp_requests (requested_at);
CREATE INDEX sessions_expires_at ON sessions (expires_at);
CREATE INDEX suggestions_reviewed_at ON suggestions (reviewed_at) WHERE status <> 'pending_review';
