# Research: Save album command — metadata providers, resilience, and storage

**Spec**: [spec.md](spec.md) | **Date**: 2026-04-23

## 1. Free / low-cost album metadata APIs (recommend 3 in cascade)

| Order | API | Free tier (summary) | Auth | Best for | Caveats |
|------:|-----|---------------------|------|----------|---------|
| **1** | **MusicBrainz** | Non-commercial use free; [JSON/XML](https://musicbrainz.org/doc/MusicBrainz_API) | **No key**; **User-Agent** must identify the app and contact | Canonical **MBIDs**, release metadata, disambiguation | **~1 request / second** per [rate limiting](https://musicbrainz.org/doc/MusicBrainz_API/Rate_Limiting); 503 when overloaded. Commercial use may need a separate [MetaBrainz](https://metabrainz.org/api) agreement. |
| **2** | **Last.fm** (Audioscrobbler) | Public API; [album.search](https://www.last.fm/api/show/album.search) | **api_key** (free registration) | “Best effort” name search, art URLs, good **relevance**-sorted lists | [Rate limit](https://www.last.fm/api/intro) enforced (error **29**); be polite with volume. |
| **3a** (pick one) | **iTunes / Apple** [Search API](https://performance-partners.apple.com/search-api) | No key in common usage for catalog search; JSON | Optional for commercial scale | **Fast** free-text match, art, UPC; simple HTTP | [Terms of use](https://www.apple.com/legal/internet-services/) apply; not for **direct** re-streaming, metadata OK for catalog. |
| **3b (alt.)** | **Deezer** [public](https://developers.deezer.com/api) | **application id**; many reads without paid account | `access_token` optional for some calls | International catalog | Rate limits; check current policy. |
| **3c (alt.)** | **Spotify Web API** [search](https://developer.spotify.com/documentation/web-api/reference/search) | Dev mode: **[free developer app](https://developer.spotify.com/)**; **Client Credentials** for server-to-server (no end-user login for search) | `client_id` + `client_secret` → short-lived access token | Rich album objects, `limit` to **10** in dev in 2026+ policy | [Development mode](https://developer.spotify.com/documentation/web-api/concepts/quota-modes) limits (users, product policy); 429 with **Retry-After**. **Plan default**: use **iTunes** as third to avoid per-user allowlists for a Telegram bot. |

**Decision (v1)**: **Primary: MusicBrainz** → **Secondary: Last.fm** → **Tertiary: iTunes Search API** (or Deezer if team prefers a single “music” brand—document in `bot` config).

**Rationale**: MusicBrainz is the strongest **open, identifier-rich** source and needs no key; Last.fm needs only a static API key; iTunes is keyless and fast for disambiguation when the first two are thin or throttled. Spotify remains an optional fourth swap-in.

---

## 2. Circuit breaker and failover (multi-provider)

**Pattern**: A **chained** `Search(query) → []Candidate` **port** in `internal/core` (interface) with **adapters** per provider. Each provider wrapped with a **per-provider circuit breaker** (e.g. [sony/gobreaker](https://github.com/sony/gobreaker) or small custom):

- On **5xx, timeout, or consecutive failures**: **Open** the breaker for a **cooldown** (e.g. 30–60s, configurable).
- While **Open**, **skip** that provider and try the next in the chain.
- On **4xx** from client (except 429 throttling if mapped): may **treat** as non-retry (no open) or open—product decision: **429** and **503** should trip toward fallback.
- **Normalize** all candidates to a shared **DomainAlbumCandidate** in core (title, artist, year, genre[], provider, providerRef).

**Spotify/Last.fm 429 / MusicBrainz 503**: do **not** mark whole stack dead—only the failing **ring**.

**Alignment with spec [FR-006](spec.md)**: if **all** providers are open or return empty, the user gets the generic “try again” / “not found” path—**no** save.

---

## 3. PostgreSQL and schema evolution

- **Target**: **PostgreSQL** 15+ (LTS) for `listeners`, `saved_albums`, and **`disambiguation_sessions`** (pending picks for **FR-009**); no Redis in v1.
- **Migrations** (versioned, repeatable in CI and prod):
  - **Tooling**: [golang-migrate/migrate](https://github.com/golang-migrate/migrate) (CLI + Go API) with SQL files, **or** [Atlas](https://atlasgo.io) with versioned HCL/SQL, **or** [pressly/goose](https://github.com/pressly/goose)—**Decision**: `golang-migrate` with files under `bot/migrations/` (or root `migrations/`) as `NNNNNN_description.{up,down}.sql` **or** single `.up.sql` per version with `migrate` convention.
- **Process**: New schema changes = **new migration number**; **no** ad-hoc `CREATE TABLE` in app startup for prod. Local `make migrate-up` / `go run` wrapper; **compose** service `postgres` with volume.
- **CI**: optional job `migrate -path ... -database $DATABASE_URL up` against ephemeral Postgres.

**Why**: Constitution **I** and **VIII** expect reproducible, reviewable schema changes; operators need a single `up`/`down` story.

---

## 4. Disambiguation state (no Redis)

- **Decision (clarified 2026-04-23)**: **Do not use Redis.** Store pending **2–3** candidate picks in **`disambiguation_sessions`** in **PostgreSQL** with `expires_at` and periodic or on-read cleanup. For **local single-process** runs, an **in-memory** map is acceptable; **production** and any **multi-replica** deployment should use **Postgres** only so all instances share state.
- **TTL**: e.g. **10–15 minutes** per row (plan-owned exact value).

---

## 5. Alternatives (rejected for v1)

| Alternative | Rejected / deferred because |
|---------------|-----------------------------|
| **Single** provider only | Fails [FR-006] resilience story and operator expectations after outage. |
| **Redis** for disambiguation | **Removed** from scope; Postgres + in-memory dev is enough; fewer moving parts in compose. |
| **Only** iTunes/Last.fm (no MusicBrainz) | Weaker canonical IDs; MusicBrainz is the standard for **non-commercial** open data. |
| **ORM-only** schema (auto-migrate) | Harder to review in PR; versioned SQL matches constitution expectations. |

---

## 6. References (external)

- MusicBrainz API & rate: https://musicbrainz.org/doc/MusicBrainz_API/Rate_Limiting  
- Last.fm album.search: https://www.last.fm/api/show/album.search  
- Apple iTunes Search API (overview): see Apple’s Search API / affiliate docs for current **legal** and **format**  
- gobreaker: https://github.com/sony/gobreaker  
- golang-migrate: https://github.com/golang-migrate/migrate
