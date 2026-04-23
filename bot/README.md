# LifeSoundtrack `bot/`

- **`internal/core`**: product strings and command semantics (no third-party chat SDKs).
- **`internal/adapter/telegram`**: first platform adapter; only this path imports `github.com/go-telegram/bot`.
- **`cmd/bot`**: process entry, config, logging, wires the active adapter.

## Develop

```bash
cd bot
go fmt ./...
go vet ./...
go test ./... -count=1
go run ./cmd/bot
```

`TELEGRAM_BOT_TOKEN` is required. At startup the process loads **`./.env`** from the **current working directory** (optional; missing file is OK). Environment variables already set in the shell **take precedence** over values in `.env`. See [`.env.example`](./.env.example), [002 quickstart](../specs/002-env-file-config/quickstart.md), and [001 runbook](../specs/001-lifesoundtrack-bot-commands/quickstart.md).

### Hot reload (optional)

With [air](https://github.com/cosmtrek/air) installed (same pin as [002 quickstart](../specs/002-env-file-config/quickstart.md)), from `bot/`:

```bash
air
```

(`air` picks up `bot/.air.toml` in the current directory, or `air -c .air.toml` explicitly.)

Watches `*.go` and `.env` under this module; CI and Docker use a normal `go build` / image build (no watcher).

## Lint (optional)

If `golangci-lint` is installed: `golangci-lint run ./...` from `bot/`.
