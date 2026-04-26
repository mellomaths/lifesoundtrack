# lifesoundtrack

Project workspace with **Spec Kit** (`.specify/`) and **Cursor** rules and skills (`.cursor/`), aligned with [`.specify/memory/constitution.md`](.specify/memory/constitution.md).

## Application: LifeSoundtrack bot (`bot/`)

The Go module at [`bot/`](bot/) implements the feature [001-lifesoundtrack-bot-commands](specs/001-lifesoundtrack-bot-commands/):

- **`bot/internal/core`** — domain command handling and all user-visible copy (platform-agnostic).
- **`bot/internal/adapter/telegram`** — first messaging adapter (long polling); the only package that imports `github.com/go-telegram/bot`.

**Run locally:** create `bot/.env` or export `TELEGRAM_BOT_TOKEN` and `DATABASE_URL` (e.g. local Postgres; `docker compose up -d` starts one), then `cd bot && go run ./cmd/bot`. **Save-album (003)** uses metadata **Spotify → iTunes → Last.fm → MusicBrainz** with per-source **`LST_METADATA_ENABLE_*`** flags: [specs/003-save-album-command/quickstart.md](specs/003-save-album-command/quickstart.md). **Save via Spotify album / share link (005):** [specs/005-save-album-spotify-url/quickstart.md](specs/005-save-album-spotify-url/quickstart.md). **List saved albums (006):** `/list` with optional artist filter; multi-page libraries use **Back/Next** buttons and/or **`/list next`** and **`/list back`** — [specs/006-list-saved-albums/quickstart.md](specs/006-list-saved-albums/quickstart.md). **Docker:** from the repo root, `docker compose up --build` runs Postgres and the bot with `AUTO_MIGRATE` (see [`compose.yaml`](compose.yaml)) — pass `TELEGRAM_BOT_TOKEN` in the environment; `DATABASE_URL` defaults for the stack when unset.

**Config file loading & dev workflow:** [specs/002-env-file-config/quickstart.md](specs/002-env-file-config/quickstart.md). **Sandbox runbook and acceptance (commands):** [specs/001-lifesoundtrack-bot-commands/quickstart.md](specs/001-lifesoundtrack-bot-commands/quickstart.md).
