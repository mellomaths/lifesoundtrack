# Research: daily recommendations (004)

**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-04-24

## 1. In-process scheduler vs external cron

- **Decision**: **`robfig/cron/v3`** inside **`bot/cmd/bot`** with `cron.WithLocation` for **`LST_DAILY_RECOMMENDATIONS_TZ`**.
- **Rationale**: One deployable; flag off → do not add entries to cron; matches **FR-010** kill switch.
- **Alternatives considered**: Separate worker or HTTP-triggered job — extra ops for v1.

## 2. Default schedule expression

- **Decision**: **`LST_DAILY_RECOMMENDATIONS_CRON`** defaults to **`0 6 * * *`** (minute 0, hour 6, every day) in the configured location.
- **Rationale**: Matches spec default **06:00**; operators override with standard cron syntax.
- **Alternatives considered**: Separate HOUR/MINUTE env vars — less flexible.

## 3. Send-then-transact ordering

- **Decision**: **Telegram success first**; then **single SQL transaction** for `last_recommended_at` + `recommendations` insert.
- **Rationale**: **FR-007** / **FR-008** — never advance rotation on failed delivery.
- **Alternatives considered**: DB-first — **rejected** (violates spec).

## 4. Post-send DB failure

- **Decision**: Log **error** with **`run_id`**, listener id, saved_album id; alert/metric optional; document manual reconciliation — user may see a pick without DB row until fixed.
- **Rationale**: Cannot enlist Telegram in a 2PC; operational rarity.
- **Alternatives considered**: Duplicate send on retry — worse than rare audit gap.

## 5. Idempotency (`run_id`)

- **Decision**: New UUID per cron tick; **`UNIQUE (listener_id, run_id)`** on **`recommendations`**.
- **Rationale**: Prevents double insert if handler retries after partial success; supports **SC-005**/**SC-006** auditing.
- **Alternatives considered**: No unique constraint — allows duplicates under retries.

## 6. Spotify URL from `saved_albums` only

- **Decision**: If **`provider_name = spotify`** and **`provider_album_id`** set → `https://open.spotify.com/album/{id}`; optional documented key in **`extra`** JSON if present; else no URL.
- **Rationale**: **A-003**; no Spotify API at send time.
- **Alternatives considered**: Live API lookup — out of scope.

## 7. Master enable flag semantics

- **Decision**: Reuse **`parseMetadataFeatureFlag`** (or extract shared **`parseOptOutBool`**) for **`LST_DAILY_RECOMMENDATIONS_ENABLE`** — **FR-012**.
- **Rationale**: One operator mental model with **`LST_METADATA_ENABLE_*`**.
- **Alternatives considered**: Inverted semantics (unset = off) — **rejected** (inconsistent).

## 8. Listener enumeration (PostgreSQL `DISTINCT` vs `ORDER BY`)

- **Decision**: Implement listener discovery with a **PostgreSQL-valid** pattern: either include **all `ORDER BY` expressions in the projected list** (then map to the domain type), or **deduplicate in an inner query / `DISTINCT ON` / `GROUP BY`** and **order only in an outer query** on columns that are legal for that shape. **Do not** use `SELECT DISTINCT` on a subset of columns while ordering by non-selected expressions (**rejects with `SQLSTATE 42P10`**). **Shipped**: **`EXISTS` subquery** on **`saved_albums`** in **`ListTelegramListenerIDsWithSavedAlbums`** (no `DISTINCT` needed).
- **Rationale**: **FR-013**–**FR-015**, spec **Defect fix: Listener enumeration**; production DB is PostgreSQL.
- **Alternatives considered**: Rely on another DB’s looser rules — **rejected** (deployment is Postgres).

## 9. Scheduler lifecycle vs Telegram long-polling

- **Decision**: Call **`cron.Cron.Start()`** in a **dedicated goroutine** started **before** **`telegram.Run`** (blocking long-poll). On process **`context` cancellation**, call **`Stop()`** and wait for graceful stop so in-flight tick work can finish per library contract.
- **Rationale**: **FR-017** — ticks must run without relying on OS cron or inbound messages; **FR-001** — daily wall-clock must advance while the bot process is up.
- **Alternatives considered**: Run Telegram and cron in separate processes — **rejected** for v1 (extra ops); external cron HTTP ping — **rejected** (new surface).

## 10. Monte Carlo tolerance for tied-tier fairness (**SC-003**)

- **Decision**: For automated harnesses with **k** tied albums and **n** i.i.d. draws, use either: **(a)** Pearson **χ²** goodness-of-fit vs uniform over **k** bins with **p ≥ 0.05**, or **(b)** per-album **binomial** 95% **Clopper–Pearson** interval that contains **n/k** for each album when **n** is large enough for asymptotics. Document which rule was used in the test output.
- **Rationale**: Removes “agreed tolerance” ambiguity in **SC-003** while staying standard for QA stats.
- **Alternatives considered**: Fixed raw count delta — **rejected** (not scale-free across **n**).
