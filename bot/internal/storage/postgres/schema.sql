CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_identities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    external_id TEXT NOT NULL,
    username TEXT,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_identities_source_external_unique UNIQUE (source, external_id)
);

CREATE INDEX IF NOT EXISTS idx_user_identities_user_id ON user_identities (user_id);

CREATE TABLE IF NOT EXISTS album_interests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    album_title TEXT NOT NULL,
    artist TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_recommended_at TIMESTAMPTZ NULL,
    CONSTRAINT album_interests_title_nonempty CHECK (length(trim(album_title)) > 0),
    CONSTRAINT album_interests_artist_nonempty CHECK (length(trim(artist)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_album_interests_user_id ON album_interests (user_id);
CREATE INDEX IF NOT EXISTS idx_album_interests_user_last_rec ON album_interests (user_id, last_recommended_at);

ALTER TABLE album_interests
    ADD COLUMN IF NOT EXISTS last_recommended_at TIMESTAMPTZ NULL;

CREATE TABLE IF NOT EXISTS recommendation_audit (
    id BIGSERIAL PRIMARY KEY,
    album_interest_id BIGINT REFERENCES album_interests (id) ON DELETE SET NULL,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    album_title TEXT NOT NULL,
    artist TEXT NOT NULL,
    recommended_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_recommendation_audit_user_time ON recommendation_audit (user_id, recommended_at DESC);
CREATE INDEX IF NOT EXISTS idx_recommendation_audit_time ON recommendation_audit (recommended_at);
