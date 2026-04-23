# Research: LifeSoundTrack — `.env` file + local hot reload (002)

**Spec**: [spec.md](spec.md) | **Date**: 2026-04-25

## 1. Open-source `.env` loading in Go

**Decision**: Use **`github.com/joho/godotenv`**.

**Rationale**

- Permissive license, **widely used**, stable API, small API surface: `Load`, `Read`, `Overload` (we **avoid** `Overload` to keep **OS env over file** for **FR-002**).
- Default behavior: **does not** override variables already set in the environment—matches the clarification session in the spec.
- Missing file: `Load` can be wrapped so “no `.env`” is acceptable when combined with an existence check, or the project can call `Load` with a path and **ignore** `os.IsNotExist` (exact pattern in implementation).

**Alternatives considered**

| Option | Outcome |
|--------|---------|
| Hand-rolled `KEY=VALUE` reader | Rejected: easy to get quotes, `export`, and comments wrong; fails **FR-007** spirit. |
| **spf13/viper** | Rejected: heavier than needed; we only need flat env-style keys for one file. |
| **kelseyhightower/envconfig** only (no file) | Rejected: does not replace `.env` file parsing. |

**Security note**

- Do not log the path’s **contents** on error. Treat parse errors as “bad config” without printing values.

## 2. Hot reload (watch and restart) for local dev

**Decision**: Use **`air`** ([`github.com/cosmtrek/air`](https://github.com/cosmtrek/air)) with a committed **`bot/.air.toml`**.

**Rationale**

- De facto standard for **Go** live rebuild + restart; supports **include_ext** for **`env`** in addition to **`go`**, so editing **`bot/.env`** triggers a restart.
- **Not** a Go module dependency: developers install the **binary**; no import into production code, keeping **`go.mod`** free of non-runtime tools if desired (only `godotenv` is a library dep).

**Alternatives considered**

| Option | Outcome |
|--------|---------|
| **reflex** / **watchexec** + `go run` | Valid; more shell glue and less Go-ecosystem convention; air bundles build + run in one. |
| **CompileDaemon** | Less active community than air; similar role. |
| **IDE-only** (GoLand / VS Code tasks) | Rejected: **FR-008** asks for a **documented** repo-native workflow, not only IDE-specific. |

**Production**

- `Dockerfile` and CI **do not** use air; they rely on a **built** binary and injected env. Aligns with Story 4 acceptance: hot reload is **not** a production concern.

## 3. File path and working directory

**Decision**: Document that **`go run` / `air`** are executed with **current working directory = `bot/`** so **`godotenv.Load(".env")`** (or `filepath.Join` with a single relative segment) resolves to **`bot/.env`**, consistent with the existing [bot/.env.example](../../bot/.env.example).

**Rationale**

- Matches existing quickstart that already `cd bot`.
- Optional follow-up: explicit `-envfile` flag (not required for v1 in spec).

## 4. Unresolved

None — all “NEEDS CLARIFICATION” from planning are closed by this document.
