# Research: Save album command — metadata providers, resilience, and storage

**Spec**: [spec.md](spec.md) | **Date**: 2026-04-24 (updated from 2026-04-23)

## 1. Metadata API chain (spec order: 2026-04-24)

| Order | API | Free tier (summary) | Auth | Best for | Caveats |
|------:|-----|---------------------|------|----------|---------|
| **1** | **Spotify** Web API [search](https://developer.spotify.com/documentation/web-api/reference/search) | Dev / server: **Client Credentials** (no user login for search) | `SPOTIFY_CLIENT_ID` + `SPOTIFY_CLIENT_SECRET` → short-lived access token | Rich album objects, popularity, markets | **429** with **Retry-After**; [quota / mode](https://developer.spotify.com/documentation/web-api/concepts/quota-modes) policy. |
| **2** | **iTunes** / Apple [Search API](https://performance-partners.apple.com/search-api) | Common usage: catalog search without key; JSON | **None** for basic search (see Apple **ToS**) | **Fast** free-text match, art | Terms of use; metadata-only use for **catalog** discovery. |
| **3** | **Last.fm** [album.search](https://www.last.fm/api/show/album.search) | Public API | `LASTFM_API_KEY` (free registration) | Relevance-sorted lists, art URLs | Rate limit; error **29** when exceeded. |
| **4** | **MusicBrainz** | Non-commercial; [JSON](https://musicbrainz.org/doc/MusicBrainz_API) | **No** key; **User-Agent** must identify app + contact | **MBIDs**, canonical metadata | **~1** **req/s**; **503** when overloaded. |

**Decision (v1, aligned with [spec.md](spec.md) FR-002)**: **Spotify** → **iTunes** → **Last.fm** → **MusicBrainz** — **fixed** order; **per-catalog** **environment** **feature** **flags**; **fallthrough** when a **ring** is **off**, returns **no** **candidates**, or hits **recoverable** **failure** + **policy** (breaker **open**, **5xx**, **throttle** as agreed).

**Rationale**: **Spec** **mandates** **Spotify** **first** and **MusicBrainz** **last**; **iTunes** and **Last.fm** stay **in** **the** **middle** for **keyless** / **key** **static** **search** **without** **blocking** on **Spotify** **app** **policy** **alone**. **Operator** can **disable** any **leg** (e.g. **no** **Spotify** **credentials** in a **given** **env**).

**Alternatives considered**: **MB-first** (older **internal** **draft**): **rejected** — **conflicts** with **product** **spec** **amendment** **2026-04-24**.

---

## 1b. Per-catalog feature flags (environment)

| Variable | Default (if unset) | Purpose |
|----------|--------------------|---------|
| `LST_METADATA_ENABLE_SPOTIFY` | **true** (enable) | When **false**, do **not** call Spotify. |
| `LST_METADATA_ENABLE_ITUNES` | **true** | When **false**, skip iTunes. |
| `LST_METADATA_ENABLE_LASTFM` | **true** | When **false**, skip Last.fm (Last.fm key may be omitted in that case). |
| `LST_METADATA_ENABLE_MUSICBRAINZ` | **true** | When **false**, skip MusicBrainz. |

**Parsing**: Treat **`true`/`1`/`yes`/`on`** (case-insensitive) as **on**; **`false`/`0`/`no`/`off`** as **off**. **Unset** = **on** to preserve **sensible** **defaults** for **dev** / **prod** **templates**.

**Decision**: **Opt-out** **flags**; **all-off** → **no** **HTTP** **metadata** **calls**; user-facing **outcome** **per** [contracts/album-command.md](contracts/album-command.md) **`provider_exhausted`**-class **message** (see **spec** **SC-006**).

---

## 2. Circuit breaker and failover (multi-provider + flags)

**Pattern**: **Chained** `Search(query) → []Candidate` **port** in `internal/core` with **adapters** per **provider**. Each **enabled** **provider** **in** **chain** **order** can be wrapped with a **circuit** **breaker** (`gobreaker` or small custom):

- **Flag** **off**: **do** **not** **invoke** **HTTP**; **do** **not** **advance** **breaker** **state** for **that** **name** (or **no-op** **wrapper**).
- On **5xx**, **timeout**, or **consecutive** **failures** **for** **invoked** **calls**: **open** **breaker** **for** **cooldown** (e.g. **30s**); **try** **next** **enabled** **provider**.
- **4xx** / **client** **errors** **(except** **throttle)**: **policy** in **orchestrator**—**429/503** **→** **fallthrough** to **next** **ring** per **spec** **resilience** **story**.
- **Normalize** all **candidates** to **shared** `AlbumCandidate` in **core** (title, artist, year, genre[], `provider`, `provider_ref`).

**Spotify 429** / **MB 503** / **Last.fm 29**: **do** **not** **treat** **entire** **stack** as **permanent** **dead**—**only** **that** **ring** unless **breaker** **opens**.

**Alignment** with [spec](spec.md) **FR-006**: If **all** **enabled** **rings** **fail** or **return** **empty** → **no** **false** **save**.

---

## 3. PostgreSQL and schema evolution

- **Target**: **PostgreSQL** **15+** for `listeners`, `saved_albums`, `disambiguation_sessions`; **no** Redis in v1.
- **Tooling**: **golang-migrate**; SQL under **`bot/migrations/`** (or path **documented** in [plan](plan.md) / [quickstart](quickstart.md)).
- **Process**: **Versioned** **migrations**; **no** ad-hoc **CREATE** in **prod** **startup**.

**Why**: Constitution **I** / **VIII**; **reproducible** **schema**.

---

## 4. Disambiguation state (no Redis)

- **Decision**: **Postgres** **`disambiguation_sessions`**; **in-memory** only for **single-process** local dev. **At** **most** **2** **distinct**-**by**-**label** **lines** + **Other** in **UI** when **true** disambig **applies** ([spec](spec.md) **FR-009**; **see** **§5** for **dedupe**).
- **TTL**: **~10–15** **minutes** (`expires_at`).

---

## 5. Equivalent user-visible labels (FR-009 / SC-007)

**Problem**: Multiple **raw** **candidates** from **one** **ring** (or merged policy) can **normalize** to the **same** **`ALBUM_TITLE | ARTIST (YEAR)`** string—**presenting** **two** **buttons** with **identical** **text** **adds** **no** **information** and **violates** **UX** **(regression)**.

**Decision**: In **core**, **after** **relevance** **order** is **known**, **collapse** **candidates** **that** **share** the **same** **formatted** **label**, **keeping** the **first** **in** **that** **order**. **Then** **apply** the **at-most-two**-**distinct**-**labels** **rule** and **whether** to **show** **Other** **(only** **when** a **real** **two**-**choice** **list** **is** **shown**).

**Rationale**: Matches **spec** **amendment** **(equivalent** **labels)**; **tie**-**break** is **deterministic** (**first** **wins**).

**Alternatives considered**: **Hide** **only** **in** **Telegram** **adapter** — **rejected**; **label** **logic** **belongs** in **domain** so **tests** and **non**-**Telegram** **hosts** **share** **behavior**.

---

## 6. Alternatives (rejected for v1)

| Alternative | Rejected / deferred because |
|-------------|----------------------------|
| **MusicBrainz-first** chain (pre-2026-04-24 draft) | **Superseded** by **spec** **amendment**: **Spotify** **first** **…** **MusicBrainz** **last**. |
| **Single** provider | **Conflicts** with **FR-002** and **resilience** **stories**. |
| **Redis** for disambiguation | **Out** of **scope**; **Postgres** **+** **optional** **in-memory** **dev**. |

---

## 7. References (external)

- Spotify Web API: https://developer.spotify.com/documentation/web-api/  
- Apple iTunes Search API / terms: see Apple’s current **Search** + **legal** **pages**  
- Last.fm `album.search`: https://www.last.fm/api/show/album.search  
- MusicBrainz rate: https://musicbrainz.org/doc/MusicBrainz_API/Rate_Limiting  
- gobreaker: https://github.com/sony/gobreaker  
- golang-migrate: https://github.com/golang-migrate/migrate

