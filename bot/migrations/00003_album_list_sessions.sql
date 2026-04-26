-- +goose Up
CREATE TABLE album_list_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listener_id UUID NOT NULL REFERENCES listeners (id) ON DELETE CASCADE,
    artist_filter_norm TEXT,
    current_page INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX album_list_sessions_listener_created ON album_list_sessions (listener_id, created_at DESC);
CREATE INDEX album_list_sessions_expires_at ON album_list_sessions (expires_at);

-- +goose Down
DROP TABLE IF EXISTS album_list_sessions;
