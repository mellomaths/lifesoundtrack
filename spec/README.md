# Spec-Driven Development (SDD)

This folder defines **what** the Telegram bot must do before (or alongside) implementation. Follow this order when adding behavior:

1. **Write or update the spec** — Master behavior in [`bot-spec.md`](bot-spec.md); per-command detail under [`commands/`](commands/).
2. **Add acceptance scenarios** — Use **Given / When / Then** tables so behavior is testable.
3. **Implement** — Handlers live in `bot/internal/handlers/`; wiring in `bot/internal/app/`, transport in `bot/internal/telegram/`, persistence in `bot/internal/storage/postgres/`, entrypoint `bot/cmd/bot/`.
4. **Test** — Table-driven tests should mirror the spec scenarios; link the spec path in test comments when useful.

## Naming

- One file per command under `spec/commands/<command>.md` (e.g. `start.md` for `/start`).
- The master registry of commands (including **TBD** placeholders) stays in [`bot-spec.md`](bot-spec.md).

## Related code

| Spec | Implementation | Tests |
|------|----------------|-------|
| [`commands/start.md`](commands/start.md) | `bot/internal/handlers/start.go` | `bot/internal/handlers/start_test.go` |
| [`commands/help.md`](commands/help.md) | `bot/internal/handlers/help.go` | `bot/internal/handlers/help_test.go` |
| [`commands/album.md`](commands/album.md) | `bot/internal/handlers/album.go` | `bot/internal/handlers/album_test.go` |
| [`commands/daily-recommendation.md`](commands/daily-recommendation.md) | `bot/internal/recommendation/run.go`, `bot/internal/storage/postgres/recommendation.go` | `bot/internal/recommendation/run_test.go` |
