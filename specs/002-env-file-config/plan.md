# Implementation Plan: LifeSoundtrack — `.env` loading and local hot reload

**Branch**: `002-env-file-config` | **Date**: 2026-04-25 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification in `specs/002-env-file-config/spec.md` plus planning notes: use an **open-source** library for `.env` and a **hot refresh** path so local edits to **Go sources** and **`.env`** trigger a **restart**.

**Note**: `setup-plan.sh` may resolve paths from the **current git branch**; the canonical feature path for this work is **`specs/002-env-file-config/`** (see `.specify/feature.json`). **Phase 2** tasks: `/speckit.tasks` (not produced by this command).

## Summary

Add **`github.com/joho/godotenv`** to load **`bot/.env`** early in `cmd/bot` **before** `config.FromEnv()`, using the process **current working directory** of `bot/` (documented in quickstart) so that:

- **Precedence** matches **FR-002** / spec clarifications: variables already in the **OS environment** are **not** overwritten by the file (godotenv default behavior: **no Overload**).
- **Missing** `.env` is **non-fatal**: skip load or treat “file not found” as a no-op so **FR-003** (env-only) still works.

For **local development hot reload** (user Story 4, **not** production), standardize on **`air`** ([`github.com/cosmtrek/air`](https://github.com/cosmtrek/air)): a **`bot/.air.toml`** (or `bot/.air.toml` + `make`/docs) that rebuilds and restarts the binary on changes to **`.go`** and **`.env`** under `bot/`. **CI and Docker** keep running the compiled binary (or `go run` once) with **no** watcher in the default image.

## Technical Context

**Language/Version**: Go **1.22+** (same as `bot/go.mod`).

**Primary dependencies (this feature)**:

- **`github.com/joho/godotenv`** — MIT, de facto standard; small surface; `Load` honors existing env (aligns with **FR-002**). Pin a **released** version in `go.mod`.
- **`air`** — installed **out of band** for developers (`go install github.com/cosmtrek/air@latest` or release binary); **not** imported by application code. Config lives in **`bot/.air.toml`** in VCS (no secrets; paths only).

**Storage**: N/A (no new persistence; `.env` is a local file on disk).

**Testing**: `go test ./...` under `bot/`; add **unit tests** for config bootstrap (e.g. with `t.Setenv` / `t.Chdir` and a `testdata/` `.env` fixture) to assert precedence and optional file; **air** is validated manually (save `.env` and a `.go` file → process restarts).

**Target platform**: Linux/macOS/Windows dev boxes for **air**; production Linux containers unchanged in behavior (env from Compose/Kubernetes).

**Project type**: Long-running **CLI** / daemon under `bot/cmd/bot`.

**Performance goals**: N/A (startup + rare reload in dev only).

**Constraints**: **Never** log token or full `.env` content (**FR-004**); do not add `godotenv` to **`internal/core`** (config/bootstrap stays in `cmd` or `internal/config` only).

**Scale/Scope**: One bot process, one documented `.env` path for local runs.

## Constitution Check

*GATE: Satisfied (post-design).*

- **Code quality**: `gofmt` / `go vet` / optional `golangci-lint`; new dependency justified in **Complexity** below; **air** is tooling-only, not a runtime import.
- **REST API**: N/A.
- **Testing**: Unit tests for **env loading + precedence**; manual or smoke for **air** in quickstart.
- **User experience / domain**: Unchanged; **FR-006** preserved — no domain/adapter string changes.
- **Monitoring / logging / performance**: N/A beyond existing slog rules; reload is dev-only.
- **Containerization**: `Dockerfile` / `compose.yaml` **unchanged in spirit**: still inject `TELEGRAM_BOT_TOKEN` etc.; do **not** require **air** in the image. Document that hot reload is **local** only.
- **Documentation accuracy**: `quickstart.md` in this spec folder; root/`bot` README link updates in tasks. **IX**: remove stale references if any (none expected for this small change).

## Project Structure

### Documentation (this feature)

```text
specs/002-env-file-config/
├── spec.md
├── plan.md                 # this file
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    └── env-loading.md
```

### Source (repository) — **touch points**

```text
bot/
├── go.mod / go.sum         # +godotenv
├── cmd/bot/main.go         # call `LoadLocalDotEnv()` (or equivalent) from `internal/config` before `FromEnv()`
├── internal/config/        # FromEnv() unchanged in signature; may accept optional test hooks
├── .env.example            # unchanged keys; note godotenv
├── .air.toml               # NEW: air build + watch (include .go + .env)
└── README.md               # "dev: air" vs "prod: go run / binary"
```

**Structure decision**: **`github.com/joho/godotenv`** is used only from **`bot/internal/config`** (e.g. `loadenv.go`); **`cmd/bot/main.go`** calls that package and does **not** import `godotenv` directly unless you later collapse to a one-liner—either pattern is fine as long as **`internal/core`** and **`internal/adapter/*`** never import it. **air** is **not** a Go import—only **`bot/.air.toml`** and docs.

## Complexity Tracking

| Violation | Why needed | Simpler alternative rejected because |
|-----------|------------|-------------------------------------|
| New third-party dep (`godotenv`) in **`internal/config`** | **FR-007** requires a maintained OSS **parser**; hand-rolling KEY=VALUE is error-prone and security-sensitive | `os.ReadFile` + manual parse duplicates battle-tested behavior |
| **air** + `.air.toml` (extra moving parts) | **FR-008** and Story 4 require **watch + restart** for **both** Go and the **`.env` file**; a single well-known tool reduces bespoke scripts | Raw `watchexec` or loop scripts are harder to make cross-OS and less idiomatic in Go community |

## Phase 0 & Phase 1 Artifacts (this command)

| Artifact | Path | Purpose |
|----------|------|---------|
| Research | [research.md](research.md) | `godotenv` vs alternatives; `air` vs alternatives; precedence & security notes |
| Data model | [data-model.md](data-model.md) | Env keys, load order, no DB |
| Contract | [contracts/env-loading.md](contracts/env-loading.md) | Precedence, file path, and bootstrap order for config |
| Quickstart | [quickstart.md](quickstart.md) | `.env` + `air` for local dev; plain env for Docker |

**Phase 2** (`tasks.md`) is **not** created by `/speckit-plan`; run **`/speckit.tasks`** next.

## Implementation notes (for tasks phase)

1. **Bootstrap order** in `main`: optional `godotenv.Load` for `bot/.env` (path from `os.Getwd()` or fixed relative path documented as “run from `bot/`”); then `config.FromEnv()`.
2. **Tests**: `t.Chdir` into a temp dir with a fake `.env` and assert `TELEGRAM_BOT_TOKEN` and precedence vs `t.Setenv`.
3. **Air**: `include_ext` includes **`go`** and **`env`**; `cmd` builds `./cmd/bot`; exclude `tmp/`, `vendor/`.
4. **No** `godotenv` in `internal/adapter/telegram` or `internal/core`.

## Branch and Spec Kit path alignment

`setup-plan.sh` / `check-prerequisites.sh` resolve `FEATURE_DIR` from the **current git branch** in some environments. The canonical feature for this work is **`specs/002-env-file-config/`** (see [`.specify/feature.json`](../../.specify/feature.json)). If your branch is still `001-…`, switch or merge before assuming scripts point at 002.
