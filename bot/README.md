# LifeSoundtrack `bot/`

- **`internal/core`**: product strings, `/album` save flow, and command semantics (no third-party chat SDKs).
- **`internal/adapter/telegram`**: first platform adapter; only this path imports `github.com/go-telegram/bot`.
- **`internal/store`**: PostgreSQL (`pgxpool`) and migrations ([Goose](https://github.com/pressly/goose)).
- **`internal/metadata`**: **Spotify → iTunes → Last.fm → MusicBrainz** (see `LST_METADATA_ENABLE_*` in [`.env.example`](./.env.example)) behind `MetadataOrchestrator`.
- **`cmd/bot`**: process entry, config, logging, wires the active adapter and database.

Feature **003** (save album): [specs/003-save-album-command/quickstart.md](../specs/003-save-album-command/quickstart.md).  
Feature **005** (paste Spotify album or share link): [specs/005-save-album-spotify-url/quickstart.md](../specs/005-save-album-spotify-url/quickstart.md).

## Develop

```bash
cd bot
go fmt ./...
go vet ./...
go test ./... -count=1
go run ./cmd/bot
```

`TELEGRAM_BOT_TOKEN` and `DATABASE_URL` are required. Optional `AUTO_MIGRATE=true` runs SQL migrations at startup; otherwise run Goose yourself against `DATABASE_URL` (migrations in [`migrations/`](./migrations/)):

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir ./migrations postgres "$DATABASE_URL" up
```

If you previously applied migrations with **golang-migrate** on this database, reset the DB or use a new database before switching: Goose tracks versions in `goose_db_version`, not `schema_migrations`.

At startup the process loads **`./.env`** from the **current working directory** (optional; missing file is OK). Environment variables already set in the shell **take precedence** over values in `.env`. See [`.env.example`](./.env.example), [002 quickstart](../specs/002-env-file-config/quickstart.md), [001 runbook](../specs/001-lifesoundtrack-bot-commands/quickstart.md), and the **003** link above.

### Hot reload (optional)

With [air](https://github.com/cosmtrek/air) installed (same pin as [002 quickstart](../specs/002-env-file-config/quickstart.md)), from `bot/`:

```bash
air
```

(`air` picks up `bot/.air.toml` in the current directory, or `air -c .air.toml` explicitly.)

Watches `*.go` and `.env` under this module; CI and Docker use a normal `go build` / image build (no watcher).

## Lint (optional)

If `golangci-lint` is installed: `golangci-lint run ./...` from `bot/`.
