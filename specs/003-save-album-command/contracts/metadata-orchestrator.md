# Contract: Metadata orchestration (chained providers + circuit breaker)

**Spec**: [spec.md](../spec.md) | **Research**: [research.md](../research.md) | **Date**: 2026-04-23

## Port (core)

```text
type MetadataOrchestrator interface {
  // Search returns 0 to many normalized candidates, ordered by relevance.
  // Implementations use provider chain: MusicBrainz -> Last.fm -> iTunes.
  // Context carries deadlines (e.g. 10s per provider total budget).
  Search(ctx context.Context, query string) ([]AlbumCandidate, error)
}
```

**Error** semantics:

- `ErrNoMatch` (sentinel) — not an infrastructure failure; **no** open breaker.
- `ErrAllProvidersExhausted` — every provider **skipped** (open) or **failed**; user message is generic.
- `context.DeadlineExceeded` — treat like transient if any provider not tried, else `ErrAllProvidersExhausted` product-side.

## `AlbumCandidate` (domain)

| Field | Notes |
|-------|--------|
| `title` | Required non-empty. |
| `primary_artist` | May be empty if provider is thin. |
| `year` | Optional. |
| `genres` | Optional slice, max 8 in app. |
| `relevance` | **Float** 0..1 for ordering; **or** keep provider order and map position to rank. |
| `provider` | `musicbrainz` \| `lastfm` \| `itunes` |
| `provider_ref` | Opaque (MBID, Last.fm name+artist key, iTunes `collectionId` string). |
| `art_url` | Optional HTTPS. |

**Cap**: Truncate to **top 3** for UI after merge **per provider** then global merge, or one search that returns 20 and slice to 3 in orchestrator (plan implementation detail).

## Provider adapter rules

- **Respect** MusicBrainz **1 rps** (in-process throttle or queue).
- **User-Agent** for MusicBrainz: `LifeSoundTrackBot/1.0 (your-contact-or-repo-url)`.
- **Last.fm / iTunes**: read keys from **env** (`LASTFM_API_KEY`, none for iTunes in basic usage) — no keys in repo.

## Circuit breaker (per provider)

- **Name**: `ProviderMusicBrainz`, `ProviderLastfm`, `ProvideriTunes` each own breaker.
- **Open** on: N consecutive errors (5xx, timeout) or explicit policy for 429/503; **Half-open** after cooldown to probe.
- **Closed** on success.
- **Logging**: log `provider=`, `state=open|closed`, **not** the API key; **not** the full response body at Info.

## Testing

- **Fakes** implement `MetadataOrchestrator` with fixture JSON.
- **Integration** (optional in CI with `-tags=integration`) hit [MusicBrainz test](https://musicbrainz.org/ws/2/) sparingly in CI or skip.

## Evolution

- Add **Spotify** as fourth ring: new adapter + `provider` value `spotify`, same `AlbumCandidate` shape.
