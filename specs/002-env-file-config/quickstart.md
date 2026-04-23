# Quickstart: `.env` + local hot reload (002)

**Plan**: [plan.md](plan.md) | **Date**: 2026-04-25

**Prerequisite**: The bot’s **domain and Telegram** behavior is unchanged from [001 — quickstart](../../001-lifesoundtrack-bot-commands/quickstart.md); this runbook only adds **file-based env** and **dev reload**.

## 1. One-time: install `air` (for Story 4 / FR-008)

```bash
go install github.com/cosmtrek/air@v1.49.0
# Ensure $(go env GOPATH)/bin is on your PATH
```

**Version pins:** **`air`** is pinned here for a reproducible dev install; the **`joho/godotenv`** version used at runtime is pinned in **`bot/go.mod`** (see [tasks](tasks.md) **T001** / implementation). Keep both pins updated together when you bump tooling. *Keep **air** on the **dev machine** only, not in the production image.*

## 2. Configure `bot/.env`

From the **repository root**:

```bash
cd bot
cp .env.example .env
# Edit .env: set TELEGRAM_BOT_TOKEN; optional LOG_LEVEL
```

**Never commit** a real `bot/.env`.

**Precedence**: values you **export** in the shell **override** the same keys in `.env` (useful for one-off tests).

**Invalid file**: a present `bot/.env` that **cannot be parsed** (e.g. badly malformed line) makes the process **exit at startup** with a **generic** log line (no token, no file dump). A **missing** or **valid** file with a **missing `TELEGRAM_BOT_TOKEN`** still fails with the same **“token required”** class of error as without any file.

## 3. Run with hot reload (local)

Still inside **`bot/`** (so `.env` resolves to `./.env`):

```bash
air
```

`air` uses **`bot/.air.toml`** (added in implementation) to:

- Rebuild `cmd/bot` on **`.go`** changes.
- Restart when **`.env`** changes **once** that file is included in the watch list.

If `air` is not installed, fall back to:

```bash
go run ./cmd/bot
```

(no automatic reload; stop/start manually).

## 4. Run without `.env` (env-only / CI / Docker)

Do **not** rely on a file: export or inject variables (e.g. Compose `environment:`, pipeline secrets). **godotenv** in `main` **must** treat a missing `bot/.env` as **non-fatal** per **FR-003**.

## 5. Docker / Compose

Unchanged: pass **environment variables** into the container; **do not** require **air** in the `Dockerfile` for default production-style runs. Optional dev-only Compose override could mount `.env` as a file—document separately if added.

## 6. Quality checks

```bash
cd bot
go test ./... -count=1
go vet ./...
```

After implementation, add tests that assert **precedence** (OS over file) and **optional file**.
