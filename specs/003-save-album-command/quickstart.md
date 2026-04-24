# Quickstart: Save album command (003) — local dev

**Plan**: [plan.md](plan.md) | **Date**: 2026-04-24

**Prereqs**: **Go 1.22+**, **Docker** and **Docker Compose** (for Postgres), Telegram **bot token** (sandbox). **Spotify** and **Last.fm** need **API** **registration** for **full** **chain**; **iTunes** and **MusicBrainz** are **keyless** (MusicBrainz: **User-Agent** only).

## 1. Environment (`.env` in `bot/` per [002 spec](../002-env-file-config/spec.md) pattern)

### Core and database

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | **Yes** | |
| `LOG_LEVEL` | No | e.g. `INFO` |
| `DATABASE_URL` | **Yes** (for save path) | `postgres://user:pass@host:5432/lifesoundtrack?sslmode=disable` |
| `AUTO_MIGRATE` | No | `true` / `1` to run migrations at startup. |
| `MIGRATIONS_PATH` | No | Default relative to `bot` or image path `/migrations`. |
| `MUSICBRAINZ_USER_AGENT` | No | If unset, use built-in **contact** string (must identify **app** for **MB**). |

### Metadata feature flags (per [spec.md](spec.md) **FR-002**)

**Default** when **unset**: **enabled** **(`true`)** for each. Set to **`false`**, **`0`**, **`no`**, or **`off`** to **disable** a **catalog** **without** **code** **change**.

| Variable | Description |
|----------|-------------|
| `LST_METADATA_ENABLE_SPOTIFY` | When **false**, do **not** call Spotify. |
| `LST_METADATA_ENABLE_ITUNES` | When **false**, do **not** call iTunes Search API. |
| `LST_METADATA_ENABLE_LASTFM` | When **false**, do **not** call Last.fm. |
| `LST_METADATA_ENABLE_MUSICBRAINZ` | When **false**, do **not** call MusicBrainz. |

If **all** **four** are **disabled**, the bot **must** **not** **claim** a **successful** **metadata** **resolution** (see [contracts/album-command.md](contracts/album-command.md) `provider_exhausted`).

### API credentials (when the matching flag is on)

| Variable | Required if | Description |
|----------|-------------|-------------|
| `SPOTIFY_CLIENT_ID` | Spotify **enabled** | [Spotify Developer Dashboard](https://developer.spotify.com/dashboard) |
| `SPOTIFY_CLIENT_SECRET` | Spotify **enabled** | Used for **Client Credentials** (server-to-server). |
| `LASTFM_API_KEY` | Last.fm **enabled** | [last.fm API account](https://www.last.fm/api) |

*If* **Spotify** *is* **enabled** *but* **credentials** *are* **missing**, *implementation* **falls* *through* *to* *iTunes* (see [plan.md](plan.md)); *check* **logs* *at* *Warning* *without* *secret* *values*.*

## 2. Start Postgres

From repo root:

```bash
docker compose up -d postgres
```

Defaults for user/password/DB are **compose**-specific; align `DATABASE_URL` (often port **5432** on the host). Example:

`postgres://lifesound:lifesound@localhost:5432/lifesound?sslmode=disable`

## 3. Migrations

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

- **Chain** **order** (for debugging): **Spotify** → **iTunes** → **Last.fm** → **MusicBrainz** — **only** **enabled** **steps** run.
- `/album …` — if **search** **returns** **several** **rows** that **all** **format** to the **same** **`Title | Artist (Year)`** string, the **bot** **should** **save** **without** asking **(first** by **relevance**)**; **if** **two** **different** **labels** are **offered**, pick **one** or **Other** (no **save** for **Other**).
- **Disable** **Spotify** **for** **testing** **later** **rings**: `LST_METADATA_ENABLE_SPOTIFY=false` (and **ensure** other **flags** **true** if you need **iTunes**/…).

## 6. When something fails

- **“Try** **again”** / **no** **candidates**: check **flags**, **Spotify/Last.fm** **keys**, **network**, **breakers** in **logs** (provider **name** **+** **error** **class** **only** at **Info**).
- **Migrations** issues: `migrate` **version** / **force** in **dev** **only** with **care**.

## 7. Acceptance smoke

- [ ] `migrate up` on empty DB.  
- [ ] `/album` with **empty** text → need **search** text.  
- [ ] All **`LST_METADATA_ENABLE_*=false`** → **no** **false** **success**; **sensible** **user** **message**.  
- [ ] **One** **happy** **save** with **`saved_albums`** **row** (`provider_name` **one** of **spotify** / **itunes** / **lastfm** / **musicbrainz`).  
- [ ] **Multi-match** (two **distinct** **labels**): two **candidates** + **Other** → **pick** **album** / **Other** per **spec**.  
- [ ] **Duplicate**-**label** **collapse** (e.g. **two** **raw** **rows**, **same** **formatted** **label**): **no** **disambig** **list**; **saves** **(first** **by** **relevance**)**.

## Cross-links

- [001 — quickstart](../001-lifesoundtrack-bot-commands/quickstart.md) — base Telegram.  
- [002 — quickstart](../002-env-file-config/quickstart.md) — `.env` and dev **reload**.

