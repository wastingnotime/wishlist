CREATE TABLE privacy_public_reviews (
  feature_id TEXT PRIMARY KEY REFERENCES features(id),
  requested_at TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'reviewed')),
  reviewed_at TIMESTAMPTZ
);
