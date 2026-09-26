CREATE TABLE apps (
  id TEXT PRIMARY KEY, slug TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
  description TEXT NOT NULL, url TEXT NOT NULL, active BOOLEAN NOT NULL
);
CREATE TABLE features (
  id TEXT PRIMARY KEY, slug TEXT NOT NULL, title TEXT NOT NULL,
  description TEXT NOT NULL, app_id TEXT NOT NULL REFERENCES apps(id),
  status TEXT NOT NULL CHECK (status IN ('voting', 'producing', 'delivered')),
  published_at TIMESTAMPTZ NOT NULL, producing_at TIMESTAMPTZ,
  delivered_at TIMESTAMPTZ, delivery_url TEXT,
  UNIQUE (app_id, slug)
);
CREATE TABLE identities (
  id TEXT PRIMARY KEY, email TEXT NOT NULL UNIQUE, created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE otp_challenges (
  id TEXT PRIMARY KEY, email TEXT NOT NULL UNIQUE, digest TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL, attempts INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE otp_requests (email TEXT NOT NULL, requested_at TIMESTAMPTZ NOT NULL);
CREATE INDEX otp_requests_email_time ON otp_requests (email, requested_at);
CREATE TABLE sessions (
  id TEXT PRIMARY KEY, identity_id TEXT NOT NULL REFERENCES identities(id),
  expires_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE votes (
  identity_id TEXT NOT NULL REFERENCES identities(id),
  feature_id TEXT NOT NULL REFERENCES features(id),
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (identity_id, feature_id)
);
CREATE INDEX votes_feature_id ON votes (feature_id);
CREATE TABLE suggestions (
  id TEXT PRIMARY KEY, app_id TEXT NOT NULL REFERENCES apps(id),
  identity_id TEXT NOT NULL REFERENCES identities(id),
  title TEXT NOT NULL, description TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending_review' CHECK (status IN ('pending_review', 'accepted', 'merged', 'rejected')),
  resulting_feature_id TEXT REFERENCES features(id),
  created_at TIMESTAMPTZ NOT NULL, reviewed_at TIMESTAMPTZ
);
CREATE INDEX suggestions_identity_time ON suggestions (identity_id, created_at);
CREATE INDEX suggestions_pending_time ON suggestions (status, created_at);
