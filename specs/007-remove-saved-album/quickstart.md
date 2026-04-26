# Quickstart: `/remove` saved album (007)

**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-04-26

## Prereqs

- Same as [001 quickstart](../001-lifesoundtrack-bot-commands/quickstart.md): **Telegram** bot token, **PostgreSQL**, **`goose`** migrations applied (through latest in `bot/migrations/`).
- **Bot** built from `bot/` (`go build -o ... ./cmd/bot`).

## Release / acceptance (SC-001 – SC-003)

[spec.md](spec.md) **SC-001**–**SC-003** are release-level (manual or scripted **acceptance**). Use the manual checks below as smoke, plus `cd bot && go test ./...` (**tasks** **T017**). Repeat for regression as needed.

## Manual checks

1. **Start** a private chat with the bot: **`/help`** — confirm **`/remove`** appears with a one-line description alongside **`/list`**, **`/album`**, etc.
2. **Save** an album (e.g. **`/album In Rainbows`**) so you have at least one **`saved_albums`** row.
3. **Remove** with matching title (try different **case** / **spaces**): **`/remove in rainbows`** — expect **success** and a follow-up **`/list`** (or query) **no longer** **shows** that row.
4. **Not found**: **`/remove Nonexistent Album Title`** — expect **not found**, **no** error implying server failure.
5. **Partial title** (e.g. saved **`Abbey Road (Remastered) — The Beatles`**, user sends **`/remove Abbey Road`**): expect a **numbered** **list** **and** **inline** **buttons** (one per option) in **Telegram** — not **text-only** disambig. **Complete** the **pick** by **tapping** a **button** (**SC-003**). **Regress** by **sending** **`1`** in **text** **instead** (must **still** work).
6. **Many partials** (optional, DB fixture): if **>3** **saved** **titles** **all** **contain** the same short query, expect the **“more** **than** **3** — **be** **more** **specific”** **message** **(no** **long** **enumeration**).
7. **Empty query**: **`/remove`** (nothing after) — expect **usage** hint, **no** delete.
8. **Duplicate** **titles** (optional): create **two** saves with the **same** **normalized** **title** (e.g. save twice in tests or DB fixture). **`/remove <title>`** should show a **numbered** **list** **with** **buttons**; use **a** **button** **or** **`1`/`2`** text and confirm **only** **one** row **removed**.

## Developer tests

```bash
cd bot && go test ./...
```

Pay attention to **`internal/adapter/telegram`**, **`internal/core`**, and **`internal/store`** packages touched by the implementation.

## Operator docs

When this feature **ships**, ensure the **root** [README.md](../../README.md) **and** this **quickstart** mention **`/remove`** in the same breath as other **commands** if the **README** lists **command** **inventory** (align with [spec.md](spec.md) **FR-008** and **clarified** help requirements).
