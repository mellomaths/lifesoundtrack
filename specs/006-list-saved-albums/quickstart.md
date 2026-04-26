# Quickstart: `/list` saved albums (manual verification)

**Plan**: [plan.md](./plan.md) | **Bot module**: `bot/`

## Prerequisites

- PostgreSQL + migrations applied (including **`00003_album_list_sessions`**).
- `DATABASE_URL`, `TELEGRAM_BOT_TOKEN` set; bot running per repo README / Compose.

## Operator reference (text paging)

Document for **contributors / operators** (see [spec.md](./spec.md) **FR-006**): when a user has **more than five** matching saves, paging works via **inline Back/Next** on Telegram **and** via **`/list next`** and **`/list back`** (uses the listener’s latest non-expired list session). The **root** [README.md](../../README.md) and [bot/README.md](../../bot/README.md) should mention these commands after implementation.

## Happy paths

1. **Empty library** (new Telegram user, never saved): send **`/list`**.  
   **Expect**: Friendly message explaining how to save with **`/album`** (and optional Spotify link example), **not** an empty bullet list.

2. **List all, one page**: save **3** albums (`/album …`), then **`/list`**.  
   **Expect**: Up to **3** lines, **`Title | Artist (Year)`**, **no** Back/Next.

3. **List all, two pages**: save **6+** distinct saves, **`/list`**.  
   **Expect**: **5** albums, **“page 1 of …”**, **Next** button; tap **Next** → remaining albums, **Back** returns to page 1.

4. **Artist filter**: saves include **Beatles** and **non-Beatles**; send **`/list Beatles`**.  
   **Expect**: Only matching **saved** rows; case/spacing variants (`beatles`, `  BEATLES  `) behave the same.

5. **No matches**: **`/list zznomatchzz`**.  
   **Expect**: **No** unrelated albums; **not** the “you have no saves yet” message if the user has other albums.

6. **Text paging**: With a multi-page list visible, send **`/list next`** (instead of tapping).  
   **Expect**: Same page as **Next** callback would show (uses latest session).

## Regression

- **`/album`** disambiguation: still pick with **`1`**/**`2`** or buttons.
- **`/help`** mentions **`/list`**.

## Automated tests

From repo root:

```bash
cd bot && go test ./...
```

**Expect**: New tests for list parsing, normalization, and store pagination pass.
