# Quickstart: Spotify album URL save (manual verification)

**Plan**: [plan.md](./plan.md) | **Bot module**: `bot/`

**In one sentence:** send `/album` with either a normal search line **or** a Spotify album page / `spoti.fi` link; the bot saves one album when the link is valid, without asking you to pick from a list.

## Two ways to use `/album`

| Listener input (conceptual) | What happens |
|----------------------------|--------------|
| **Free text** — title, artist, informal phrase (same as before) | **FREE_TEXT** path: catalog **search** + possible **pick** among albums when needed. |
| **Spotify album page link** or **Spotify share short link** (`spoti.fi`, etc.) | **SPOTIFY_URL** / **SHORT_URL** path: **direct** album save when the link resolves; **no** “pick an album” step for that command when one album is found. |

## Performance (what to expect)

[spec.md](./spec.md) targets **about fifteen seconds** for a **typical** successful link save when Spotify, the database, and the network are healthy—the whole command, not a single sub-step. The implementation plan uses a **shorter** timeout for **short-link** redirect work alone so that step does not eat the full user-visible budget; see [plan.md](./plan.md) **Performance Goals**.

## Prerequisites

- PostgreSQL reachable; `DATABASE_URL` set  
- `TELEGRAM_BOT_TOKEN` set  
- Spotify **enabled** with valid `SPOTIFY_CLIENT_ID` / `SPOTIFY_CLIENT_SECRET` (`LST_METADATA_ENABLE_SPOTIFY` true or default-on per 003)  
- Bot running locally or in Docker per repo README / `bot/Dockerfile`

## Happy path — full URL

In private chat with the bot:

```text
/album https://open.spotify.com/intl-pt/album/1fneiuP0JUPv6Hy78xLc2g?si=dummy
```

**Expect**: Single success message (“Saved: …”) **without** a “Pick the album” prompt. Row in `saved_albums` with `provider_name` spotify and matching `provider_album_id`.

**Optional** (embedded link): `/album hey listen https://open.spotify.com/intl-pt/album/1fneiuP0JUPv6Hy78xLc2g?si=dummy` — **expect** the **same** direct-link success when the link is **unambiguous** (see [spec.md](./spec.md) Edge Cases).

## Happy path — short link

Paste a `https://spoti.fi/...` link that opens an album in the Spotify app/web.

**Expect**: Same as full URL after redirect resolution; no disambiguation.

## Regression — free text

```text
/album Abbey Road Beatles
```

**Expect**: Existing behavior (search + possible disambiguation); unchanged.

## Failure paths

1. **Spotify disabled** (direct-link path): Turn off `LST_METADATA_ENABLE_SPOTIFY`. For a pasted **open.spotify.com** album URL or supported short link, expect coherent **non-success**, **no** save, **no** silent fallback to **searching** the raw URL string. **FREE_TEXT** may still use other enabled catalogs.  
2. **Generic non-Spotify URL**: `/album https://example.com/foo` — this is **FREE_TEXT** ([spec.md](./spec.md) **FR-008** / **FR-004**): expect normal **search** behavior on the **full** string (often **no** useful match—not the “couldn’t use that link” path reserved for **failed** Spotify direct links).  
3. **Bad Spotify album / short link**: malformed share, redirect to non-album, or **`open.spotify.com/track/...`** — expect bad-link or not-found for the **direct-link** path, **no** save, **no** `Search` fallback on the raw URL for that attempt.

## Automated tests

From repo root:

```bash
cd bot && go test ./...
```

Add/extend tests under `internal/core` and `internal/metadata` per [research.md](./research.md) §7.
