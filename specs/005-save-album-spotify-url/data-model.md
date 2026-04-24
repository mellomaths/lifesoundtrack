# Data model: Spotify album URL save (005)

**Spec**: [spec.md](./spec.md) | **Date**: 2026-04-24

## Persistence

No **new** tables or migrations are required for 005.

| Table / object | Change |
|----------------|--------|
| `listeners` | Unchanged; still upserted on each `/album` turn. |
| `saved_albums` | Unchanged schema; `user_query_text` continues to store the **user’s** full argument (**FREE_TEXT** query, pasted link alone, or link **with** short surrounding prose per [spec.md](./spec.md) scenario 5) for traceability. |
| `disambiguation_sessions` | Unchanged; **direct Spotify album path** does not create a session when resolution yields exactly one candidate. |

## Domain objects

### `AlbumCandidate` (existing)

Still the single normalized shape saved via `InsertSavedAlbum`. For the link path, the candidate comes from **Spotify album JSON** mapping (same fields as search-derived candidates: title, primary artist, year, genres, art, `provider`=`spotify`, `provider_ref`=album id).

### Parsed link (ephemeral)

Not stored. Implementation may use a small value type:

- `SpotifyAlbumID` — non-empty string validated after parse/redirect
- `LinkKind` — `full_open` | `short_spoti` (optional, for logging/metrics only; maps to spec **SPOTIFY_URL** / **SHORT_URL**)

## Validation rules

- **Album ID** must be obtained only from **trusted** parse/redirect paths ([research.md](./research.md) §1–§2).
- **No save** without a resolved `AlbumCandidate` and listener (unchanged from 003).

## State transitions

`/album` message:

1. **FREE_TEXT** ([spec.md](./spec.md) **FR-004** when not **FR-008**-eligible) → existing `Search` + optional disambiguation → save  
2. **SPOTIFY_URL** / **SHORT_URL** (**FR-008**-eligible, successful resolution, including **unambiguous** embedded link in prose) → Spotify album by id → save **or** no match / error **without** disambiguation  
3. **Multiple** qualifying links / **no** clear **primary** → user message only; **no** save (**spec** Edge Cases)

No change to disambiguation state machine except that the link branch **does not** open a session when successful.
