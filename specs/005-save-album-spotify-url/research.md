# Research: Save album via Spotify album URL and short links

**Feature**: [spec.md](./spec.md) | **Date**: 2026-04-24

## 0. Spec vocabulary ↔ implementation

**Decision**: Align code and tests with the spec’s **parameter kinds**: **FREE_TEXT** → always the existing `Search(ctx, query)` path with the **full** argument; **SPOTIFY_URL** / **SHORT_URL** → **direct-link** branch (**FR-001**–**FR-003**, **FR-005**, **FR-007**, **FR-008**). **FR-000** is the umbrella requirement that **both** families coexist on the same command.

**Rationale**: Matches the spec table **Save-album command: two argument types**; avoids conflating “URL-shaped” with “direct link.”

## 1. Detecting full `open.spotify.com` album URLs

**Decision**: Parse user text (trimmed) with a small, tested normalizer: discover **candidate** `http(s)://` URLs in the argument (not only the first token—surrounding prose may precede the link per [spec.md](./spec.md) **User Story 1** scenario 5). For each candidate, run `url.Parse`, require `host` equal to `open.spotify.com` (case-insensitive), path matches **`/album/{id}`** after ignoring empty segments (supports `/intl-pt/album/{id}` and similar locale prefixes). Album **id**: Spotify base-62 id (plan: match `[0-9A-Za-z]{10,}` or stricter per API docs in implementation; reject empty). **Primary** selection when **multiple** **FR-008** album/short targets appear follows **Edge Cases** (clear primary vs **multiple_qualifying_links**).

**Rationale**: Matches spec examples and embedded-link acceptance; avoids treating arbitrary paths as albums.

**Alternatives considered**: Regex on raw string without `url.Parse` (rejected: harder to reason about encoded characters); support `spotify:album:` URI scheme only (rejected: users paste HTTPS); “first URL only” (rejected: wrong when a non-Spotify URL precedes the album link in the same argument).

## 2. Spotify short links (`spoti.fi`)

**Decision**: If the first URL’s host is **`spoti.fi`** (and optionally `www.spotify.com` share hosts if needed later), perform **GET** with `http.Client` configured to follow redirects with a **custom `CheckRedirect`**: allow at most **5** hops; each `Location` must be **https**; each hop’s host must be on an **allowlist**: `spoti.fi`, `open.spotify.com`, `www.spotify.com`, `accounts.spotify.com` (last only if Spotify emits it in practice—tune in implementation). Stop when the final URL satisfies the **full album URL** rule in §1; extract **id** from that URL. Use a **dedicated** short client timeout (**4s**) and a small response body limit for the initial short-link hop.

**Rationale**: Spec **FR-007** requires bounded, trusted resolution; `spoti.fi` is the common user-facing shortener.

**Alternatives considered**: Third-party unshorten APIs (rejected: privacy + dependency); no short links (rejected: spec clarification requires them); follow unlimited redirects (rejected: abuse risk).

## 3. Generic HTTP(S) URLs vs failed **Spotify** direct-link attempts

**Decision** (aligned with **FR-008** / **FR-004** / **FR-005**): **Non-Spotify** hosts (e.g. `example.com`) are **not** **direct-link**-eligible: run the **normal** `Search(ctx, fullURLString)` free-text path with the **full** user argument. Only when the input is **FR-008**-**qualifying** but parsing, redirect resolution, or album lookup **fails** should the product return **bad-link**-style copy and **must not** use `Search` on the raw URL as a **substitute** for that **failed** direct attempt.

**Rationale**: Avoids misclassifying generic links as “broken Spotify shares”; still prevents silent fallback to text search for **failed** Spotify album/short-link resolution.

**Alternatives considered**: Treat every `http(s)://` as bad-link when not Spotify (rejected: blocks odd but valid free-text experiments and diverges from **FR-004**).

## 4. Metadata: single-album fetch vs search chain

**Decision**: Extend `core.MetadataOrchestrator` with:

`LookupSpotifyAlbumByID(ctx context.Context, albumID string) ([]AlbumCandidate, error)`  
and `ResolveSpotifyShareURL(ctx context.Context, shareURL string) (albumID string, err error)` (short-link resolution lives in `metadata`, not `core`).

- Implemented only on `metadata.Chain`: if Spotify flag **off** or creds missing → return **`ErrAllProvidersExhausted`** or **`ErrNoMatch`** consistent with existing ring semantics (pick one: **`ErrAllProvidersExhausted`** when disabled aligns with “configuration” messaging; **`ErrNoMatch`** when enabled but 404—**“could not find album”** copy).
- On success return **exactly one** candidate in the slice (or empty + `ErrNoMatch` if API returns empty).
- Reuse Spotify token + breaker + HTTP patterns from `runSpotifySearch`.

**Rationale**: Spec requires **direct** catalog lookup by id; other chain providers cannot resolve Spotify album IDs from pasted links reliably.

**Alternatives considered**: Encode `albumID` into `Search` query with magic prefix (rejected: opaque, error-prone); call iTunes by scraped title (rejected: breaks single-candidate guarantee).

## 5. Integration point in `SaveService`

**Decision**: In `ProcessAlbumQuery`, after empty/length checks and `UpsertListener`, **before** `Search`:

1. If **multiple** **FR-008** targets with **no** clear **primary** → **one-link** user message; **no** save (**spec** Edge Cases).
2. If `TryParseSpotifyAlbumURL(q)` → `id`
3. Else if short-link host matches supported set → `s.Search.ResolveSpotifyShareURL(ctx, url)` → `id` (or error / timeout → **bad-link** path; **no** `Search` **fallback** on raw URL)
4. If `id` known → `LookupSpotifyAlbumByID` → single-candidate `persistSave` **without** disambiguation; failures map like today.
5. Else → `s.Search.Search(ctx, q)` **including** **generic** `http(s)://` on non-Spotify hosts (**FR-004**).

**Rationale**: Minimal change to Telegram adapter; eligibility matches **FR-008**.

## 6. User-visible copy

**Decision**: Add `badLinkCopy()` for unsupported/malformed links; extend `emptyAlbumQueryCopy()` / `core` help in [copy.go](../../bot/internal/core/copy.go) to mention **paste a Spotify album link or share link**. Keep confirmations identical to current save success format.

**Rationale**: Meets **SC-004** and spec **User Story 3**.

## 7. Testing matrix (implementation)

**Decision**: Table tests: full URL with locale + `si=`; bare path; **short** surrounding prose + **one** embedded album URL (or short link) → same direct-link path as lone URL; `spoti.fi` redirect fixture (httptest server); too many redirects; redirect to non-album; Spotify disabled; Spotify 404; success 200 with fixture JSON; **generic** `https://example.com/...` → `Search` called with full string; **multiple** qualifying links → no save; **one** isolated short-link → save path with fakes (**spec** Testing NFR); isolated e2e-style persist with fakes per **T016**.

**Rationale**: Covers spec **SC-001**–**SC-003** and **FR-008** branches.
