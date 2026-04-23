-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE listeners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL,
    external_id TEXT NOT NULL,
    display_name TEXT,
    username TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT listeners_source_external UNIQUE (source, external_id)
);

CREATE TABLE saved_albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listener_id UUID NOT NULL REFERENCES listeners (id) ON DELETE CASCADE,
    user_query_text TEXT,
    title TEXT NOT NULL,
    primary_artist TEXT,
    year INT,
    genres TEXT[],
    provider_name TEXT NOT NULL,
    provider_album_id TEXT,
    art_url TEXT,
    extra JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX saved_albums_listener_id ON saved_albums (listener_id);
CREATE INDEX saved_albums_created_at ON saved_albums (created_at DESC);

CREATE TABLE disambiguation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listener_id UUID NOT NULL REFERENCES listeners (id) ON DELETE CASCADE,
    candidates JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX disambiguation_sessions_listener_id ON disambiguation_sessions (listener_id);
CREATE INDEX disambiguation_sessions_expires_at ON disambiguation_sessions (expires_at);

-- +goose Down
DROP TABLE IF EXISTS disambiguation_sessions CASCADE;
DROP TABLE IF EXISTS saved_albums CASCADE;
DROP TABLE IF EXISTS listeners CASCADE;
