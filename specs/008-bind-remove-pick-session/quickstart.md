# Quickstart: Bind remove picks to disambiguation session

**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

## Verify locally

1. From repo root, ensure Postgres matches project migrations (see root `README` / existing bot docs).
2. `cd bot && go test ./...` — all packages green after implementation.

## Manual sanity (Telegram or mental model)

1. Save several albums; run `/remove <query>` so **two** albums match (disambig with inline buttons).
2. **Without** tapping, run **another** `/remove` (different query) to get a **second** disambig with different albums (first session superseded in DB).
3. Return to the **older** message and tap a button.
   - **Expected (008):** Either removal matches the **old** list’s label if that session were still open, or a **clear non-destructive** message; **not** a removal of an album that appears **only** on the **new** list’s first row.
4. **Common path:** Single disambig, tap or type `1` — still succeeds ([SC-003](spec.md)).

## Developer scenarios (automated)

- Store test: `Open…ForListener` (or final method name) returns no row for wrong listener id, wrong session id, or expired `expires_at`.
- Core test: mock store returns two different payloads; callback path with `sessionID=S1` + index must only touch S1’s candidates.

## References

- [007 remove command](../007-remove-saved-album/contracts/remove-command.md) — base grammar and JSON shape.
- [research.md](./research.md) — rationale for callback vs text paths.
