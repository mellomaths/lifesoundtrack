-- +goose Up
ALTER TABLE saved_albums
    ADD COLUMN last_recommended_at TIMESTAMPTZ;

CREATE INDEX saved_albums_listener_last_rec
    ON saved_albums (listener_id, last_recommended_at NULLS FIRST);

CREATE TABLE recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL,
    listener_id UUID NOT NULL REFERENCES listeners (id) ON DELETE CASCADE,
    saved_album_id UUID NOT NULL REFERENCES saved_albums (id) ON DELETE CASCADE,
    title_snapshot TEXT NOT NULL,
    artist_snapshot TEXT,
    year_snapshot INT,
    spotify_url_snapshot TEXT,
    sent_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT recommendations_listener_run UNIQUE (listener_id, run_id)
);

CREATE INDEX recommendations_listener_sent ON recommendations (listener_id, sent_at DESC);

-- +goose Down
DROP INDEX IF EXISTS recommendations_listener_sent;
DROP TABLE IF EXISTS recommendations;
DROP INDEX IF EXISTS saved_albums_listener_last_rec;
ALTER TABLE saved_albums DROP COLUMN IF EXISTS last_recommended_at;
