# Quickstart: Save album command (003) — local dev

**Plan**: [plan.md](plan.md) | **Date**: 2026-04-23

**Prereqs**: **Go 1.22+**, **Docker** and **Docker Compose** (for Postgres), Telegram **bot token** (sandbox). Optional: **Last.fm** API key for the full 3-link chain; chain still works for MusicBrainz **only** in theory—configure key for best results.

## 1. Environment (`.env` in `bot/` per [002 spec](../002-env-file-config/spec.md) pattern)

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | **Yes** | |
| `LOG_LEVEL` | No | e.g. `INFO` |
| `DATABASE_URL` | **Yes** (for save path) | `postgres://user:pass@host:5432/lifesoundtrack?sslmode=disable` |
| `LASTFM_API_KEY` | **Recommended** | From [last.fm API account](https://www.last.fm/api); enables second ring. |
| `AUTO_MIGRATE` | No | `true` / `1` to run `migrations/` at startup. |
| `MIGRATIONS_PATH` | No | Absolute or relative path; default in dev is the `migrations` folder if you `cd bot`. The Docker image sets `/migrations`. |
| `MUSICBRAINZ_USER_AGENT` | No | If unset, the bot uses a built-in contact string. |

*MusicBrainz and iTunes need no key in the default design; iTunes is subject to Apple’s terms (metadata only for catalog search). **Disambiguation** state for multi-match flows lives in **PostgreSQL** (`disambiguation_sessions`) in production; no Redis.*

## 2. Start Postgres

From repo root:

```bash
docker compose up -d postgres
```

The stack uses defaults `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` (all default to `lifesound` unless you override in `.env`). Exposed port **5432** on the host. Example `DATABASE_URL` from the host:

`postgres://lifesound:lifesound@localhost:5432/lifesound?sslmode=disable`

## 3. Migrations

Install [golang-migrate](https://github.com/golang-migrate/migrate#cli) or use the Go wrapper from `bot/`.

```bash
export DATABASE_URL="postgres://..."
migrate -path bot/migrations -database "$DATABASE_URL" up
```

## 4. Run the bot

```bash
cd bot
go run ./cmd/bot
```

## 5. Try in Telegram (private chat)

- `/album Abbey Road` — expect one or more candidates, pick with button or `1` / `2` / `3`.
- If **no** `LASTFM_API_KEY`, the orchestrator may still return results from **MusicBrainz** and **iTunes** depending on query.

## 6. When something fails

- **“Try again”** on all metadata dead: check logs for **which** provider opened the breaker; verify network and rate limits.
- **Migrations** out of order: `migrate ... force` is last resort; prefer fixing version table in dev only.

## 7. Acceptance smoke

- [ ] `migrate up` clean on empty DB.  
- [ ] `/album` with empty text → “need text.”  
- [ ] `/album` with nonsense → “not found” (or equivalent).  
- [ ] At least one **successful** save with row in `saved_albums`.  
- [ ] (Optional) Two candidates → disambig → save correct row.

## Cross-links

- [001 — quickstart](../001-lifesoundtrack-bot-commands/quickstart.md) — base Telegram / domain.  
- [002 — quickstart](../002-env-file-config/quickstart.md) — `.env` and **air** for dev reload.
