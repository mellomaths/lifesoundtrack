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

`TELEGRAM_BOT_TOKEN` is required. See [`.env.example`](./.env.example) and the feature [quickstart](../specs/001-lifesoundtrack-bot-commands/quickstart.md).

## Lint (optional)

If `golangci-lint` is installed: `golangci-lint run ./...` from `bot/`.
