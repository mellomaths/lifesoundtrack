# Contract: `/album` — Spotify album URL and short-link direct path

**Spec**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md) | **Date**: 2026-04-24 (amended: **FR-008** classification)  
**Baseline**: Extends [specs/003-save-album-command/contracts/album-command.md](../../003-save-album-command/contracts/album-command.md) and [metadata-orchestrator.md](../../003-save-album-command/contracts/metadata-orchestrator.md).

**FR-000**: The save-album command serves **FREE_TEXT** (existing **003** `Search` / disambiguation) **and** **SPOTIFY_URL** / **SHORT_URL** (this contract’s direct-link classes).

## Input classification (core)

Given trimmed `query` from `ParseAlbumLine` / Telegram (may include **short** surrounding prose with **one** **unambiguous** embedded link per [spec.md](../spec.md) **User Story 1** scenario 5). **Eligibility** = [spec.md](../spec.md) **FR-008**. Parsing discovers **FR-008** targets within the full string; **primary** link rules apply when more than one qualifies.

| Class | Detection (summary) | Next step |
|-------|---------------------|-----------|
| **empty** | Same as 003 | `empty_query` — no metadata |
| **too long** | Same as 003 (`MaxQueryRunes`) | `too_long` |
| **spotify_album_direct** | Parsed `open.spotify.com` album URL → `albumID` | `LookupSpotifyAlbumByID` |
| **spotify_short_link** | Host in supported short-link set → `ResolveSpotifyShareURL` → album URL → `albumID` | `LookupSpotifyAlbumByID` |
| **spotify_link_unusable** | **FR-008**-qualifying but parse/redirect/lookup fails or final page is not an album | Bad-link user message; **no** `Search` on raw URL as **fallback** |
| **multiple_qualifying_links** | >1 distinct **FR-008** album/short target, no clear **primary** per plan | Ask for **one** link; **no** save |
| **generic_http_url** | `http(s)://` on a **non**-**FR-008** host | `Search(ctx, query)` with **full** string |
| **free_text** | Default | Existing `Search(ctx, query)` chain |

## Orchestrator port extension

```text
type MetadataOrchestrator interface {
  Search(ctx context.Context, query string) ([]AlbumCandidate, error)
  LookupSpotifyAlbumByID(ctx context.Context, albumID string) ([]AlbumCandidate, error)
  ResolveSpotifyShareURL(ctx context.Context, shareURL string) (albumID string, err error)
}
```

**Semantics**:

- **Pre**: `albumID` non-empty; ctx carries deadline.
- **Post**: Returns **0 or 1** candidate (never 2+ for success path).  
- **Errors**: Same sentinels as 003 where applicable (`ErrNoMatch`, `ErrAllProvidersExhausted`, wrapped errors for transient → mapped in `SaveService` like `Search`).

**Spotify off / no creds**: Treat as **not available** for this lookup (aligned with chain policy in implementation plan).

## Outcomes vs 003 album-command contract

| 003 outcome | 005 direct-link path |
|-------------|----------------------|
| `candidates` (disambig) | **Does not apply** when `LookupSpotifyAlbumByID` returns exactly one candidate |
| `single_match` / `single_effective_match` | **Effectively** always this when lookup succeeds (one candidate) |
| `no_match` | Spotify 404 or empty mapping |
| `provider_exhausted` | Spotify disabled / unusable for lookup |
| `saved` | Same persistence rules as 003 |

## Telegram mapping

Unchanged command: `/album <argument>`. No new buttons for the happy path.

## Acceptance

Implements spec **FR-000**–**FR-008**, **SC-001**–**SC-005**, and edge cases for redirect bounds + allowlist.
