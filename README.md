# lifesoundtrack

Project workspace with **Spec Kit** (`.specify/`) and **Cursor** rules and skills (`.cursor/`), aligned with [`.specify/memory/constitution.md`](.specify/memory/constitution.md).

## Application: LifeSoundtrack bot (`bot/`)

The Go module at [`bot/`](bot/) implements the feature [001-lifesoundtrack-bot-commands](specs/001-lifesoundtrack-bot-commands/):

- **`bot/internal/core`** — domain command handling and all user-visible copy (platform-agnostic).
- **`bot/internal/adapter/telegram`** — first messaging adapter (long polling); the only package that imports `github.com/go-telegram/bot`.

**Run locally:** create `bot/.env` or export `TELEGRAM_BOT_TOKEN`, then `cd bot && go run ./cmd/bot`. **Docker:** from the repo root, `docker compose build bot && docker compose up bot` (see [`compose.yaml`](compose.yaml)) — the image does not rely on a local `.env` file; pass env at runtime as usual.

**Config file loading & dev workflow:** [specs/002-env-file-config/quickstart.md](specs/002-env-file-config/quickstart.md). **Sandbox runbook and acceptance (commands):** [specs/001-lifesoundtrack-bot-commands/quickstart.md](specs/001-lifesoundtrack-bot-commands/quickstart.md).
