# Contract: Metadata orchestration (chained providers + circuit breaker + feature flags)

**Spec**: [spec.md](../spec.md) | **Research**: [research.md](../research.md) | **Date**: 2026-04-24

## Port (core)

```text
type MetadataOrchestrator interface {
  // Search returns 0 to many normalized candidates, ordered by relevance (for the active provider step).
  // Implementations use a fixed provider chain (see order below). Context carries deadlines
  // (e.g. ~10s total budget for metadata per spec).
  Search(ctx context.Context, query string) ([]AlbumCandidate, error)
}
```

**Chain order** ([spec](../spec.md) **FR-002**): **(1) Spotify** → **(2) iTunes** → **(3) Last.fm** → **(4) MusicBrainz**. **Skip** any **catalog** whose **`LST_METADATA_ENABLE_*`** flag is **off** (see [research.md](../research.md) §1b). **Do** **not** **HTTP** **call** **skipped** **rings**.

**Stop / fallthrough (v1 implementation)**: **Try** **enabled** **rings** **in** **order**. **Return** **candidates** from the **first** **ring** that **returns** **≥1** **normalized** **candidate** after **that** **ring’s** **search** (then **apply** **`capTop2`** for **disambig** **UI** **ceiling**). If a **ring** **returns** **no** **candidates** or is **skipped** by **flag** or **fallthrough** **error** **policy**, **continue** to the **next** **enabled** **ring**. *Merging* candidates *across* multiple rings into one list is **not** *required* for v1 unless the product changes; the spec allows plan-owned merge; the current [plan.md](../plan.md) uses “first non-empty ring wins” for latency.

**Error** semantics:

- `ErrNoMatch` (sentinel) — **no** **acceptable** **candidates** after **trying** **all** **enabled** **rings** **(or** **no** **ring** **enabled** **and** **no** **candidates)**, **without** **total** **infrastructure** **meltdown** **distinction** in **this** **layer**; **map** to **“not** **found**” **user** **copy** **where** **appropriate**.
- `ErrAllProvidersExhausted` — **all** **rings** **skipped** (flags), **all** **breakers** **open**, or **equivalent** **unrecoverable** **chain** **state**; user message is **generic** **(try** **again** / **config)**.
- `context.DeadlineExceeded` — **treat** as **transient** / **exhausted** per **orchestrator** **policy**; **no** **false** **save**.

## `AlbumCandidate` (domain)

| Field | Notes |
|-------|-------|
| `title` | Required non-empty. |
| `primary_artist` | May be empty if provider is thin. |
| `year` | Optional. |
| `genres` | Optional slice, max 8 in app. |
| `relevance` | **Float** 0..1 for ordering; **or** keep provider order and map position to rank. |
| `provider` | `spotify` \| `itunes` \| `lastfm` \| `musicbrainz` |
| `provider_ref` | Opaque (Spotify **album** **id**, iTunes `collectionId`, Last.fm name+artist key, MusicBrainz **MBID**). |
| `art_url` | Optional HTTPS. |

**Dedupe (same user-visible label)**: **Before** **capping** for **UI**, **map** each **candidate** to **`ALBUM_TITLE | ARTIST (YEAR)`** (shared **formatting** **rules** with **core**/adapter); **remove** **duplicates** **by** **identical** **string**, **retaining** **first** in **relevance** **order** (see [research.md](../research.md) §5). **If** **at** **most** **one** **row** **remains** after **dedupe** (including **several** **raw** **rows** that **all** **map** to **one** **label**), **the** **product** **does** **not** show a two-line + **Other** **prompt** for that **turn**; **core** **routes** to a **direct** **save** of the **kept** **candidate** per [album-command.md](album-command.md) **single_effective_match**.

**Cap (UI / session)**: On the **deduped** **list**, if **two** **or** **more** **distinct** **labels** **remain**, take **at** **most** **2** **(top** **by** **relevance**)** for** the disambiguation prompt (**FR-009**). **Other** is **not** an `AlbumCandidate` row; **UI** **adds** it.

## Provider adapter rules

- **Spotify**: **Client** **Credentials** **flow**; **cache** **access** **token** **in-memory** **(mutex)**; **User-Agent** as **per** **Spotify** **docs**.
- **iTunes** / **Last.fm**: as **in** [research.md](../research.md).
- **MusicBrainz**: **1** **rps** **throttle**; **User-Agent** `LifeSoundTrackBot/1.0 (contact-or-repo-url)`.
- **Secrets** from **env** only—**not** in **repo** **or** **logs** at **Info**.

## Circuit breaker (per provider)

- **Names**: `ProviderSpotify`, `ProviderItunes`, `ProviderLastfm`, `ProviderMusicBrainz` (or **aligned** **slugs**).
- **Open** on: **N** **consecutive** **failures** (5xx, timeout) per **settings**; **429/503** **→** **fallthrough** or **open** per **[plan](../plan.md)**.
- **If** **flag** **off**: **no** **invocation**—**no** **false** **“open”** from **disabled** **ring**.

## Testing

- **Fakes** implement `MetadataOrchestrator` with **fixture** **JSON** **and** **flag** **overrides** **(table-driven)**.
- **Cases**: **all** **flags** **off**; **only** **MusicBrainz** **on**; **Spotify** **fails** / **empty**, **iTunes** **succeeds**.

## Evolution

- **v1** **chain** and **flags** are **in** this **contract**; **future** **merge** **across** **providers** **or** **extra** **rings** = **amend** **this** **file** **+** **spec** **if** **user-visible** **changes**.
