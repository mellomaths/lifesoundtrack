# lifesoundtrack

The soundtrack for your life.

Behavior and operational settings are documented under [`spec/`](spec/).

## Local Postgres

Start PostgreSQL (and optional pgAdmin) with Docker:

```bash
docker compose up -d
```

### pgAdmin (optional)

Open **http://localhost:5050** and sign in (default **`admin@example.com`** / **`admin`**, or set **`PGADMIN_DEFAULT_EMAIL`** and **`PGADMIN_DEFAULT_PASSWORD`** in your environment / **`.env`** for Docker Compose). Register a server:

- **Host:** `postgres` (Docker service name, not `localhost`)
- **Port:** `5432`
- **Username:** `lifesoundtrack`
- **Password:** `lifesoundtrack`
- **Database:** `lifesoundtrack`

Use a connection string such as:

`postgres://lifesoundtrack:lifesoundtrack@localhost:5432/lifesoundtrack?sslmode=disable`

Copy [`bot/.env.example`](bot/.env.example) to **`bot/.env`** and set secrets there (including **`DATABASE_URL`**). The binary loads **`./.env`** and, if you run from the repo root, **`bot/.env`**, before reading configuration (existing environment variables still win).

## Daily recommendations

Once per UTC day at **`DAILY_RECOMMENDATION_HOUR_UTC`** (default **9**), the bot sends each user who has saved albums one **fair-random** pick and logs it in **`recommendation_audit`**. Set **`DAILY_RECOMMENDATION_ENABLED=false`** to turn this off. See [`spec/commands/daily-recommendation.md`](spec/commands/daily-recommendation.md).

## Run the Telegram bot

All Go module files live under **`bot/`**. From the repository root:

```bash
go -C bot run ./cmd/bot
```

Or:

```bash
cd bot && go run ./cmd/bot
```

## Docker image (bot)

Create **`bot/.env.production`** (for example from [`bot/.env.example`](bot/.env.example)) with the values you want inside the image. The [`bot/Dockerfile`](bot/Dockerfile) copies that file to **`.env`** in the container’s working directory so the bot loads it at startup. The file is not in git; you only need it on the machine that runs `docker build`.

Build from the repository root (build context is the **`bot/`** module). On an Intel/AMD PC this produces **linux/amd64** only; a Raspberry Pi (64-bit) needs **linux/arm64** (see below).

```bash
docker build -f bot/Dockerfile -t lifesoundtrack-bot:latest bot
```

## Deploy to a local Raspberry Pi registry

Images built on a typical PC are **amd64**. The Pi runs **arm64**, so publish an ARM image (or a multi-arch manifest). The [`bot/deploy.sh`](bot/deploy.sh) script builds with **Docker Buildx** for **`--platform linux/arm64`**, loads the image into Docker (**`--load`**), then **`docker push`**es to your registry (default **`192.168.1.100:5000`**), so insecure HTTP registries work without BuildKit quirks. Multi-arch uses **`buildx --push`** directly.

```bash
./bot/deploy.sh
# or from repo root: ./deploy.sh
```

Override registry, tag, or platform when needed:

```bash
REGISTRY=192.168.1.50:5000 TAG=v1 ./bot/deploy.sh
PLATFORM=linux/amd64,linux/arm64 ./bot/deploy.sh   # one tag, Intel + Pi
```

### Push fails: `server gave HTTP response to HTTPS client`

**Cause:** Something is speaking **HTTPS** to a registry that only serves **HTTP**.

**1 — Engine insecure list (still required on the machine that pushes):** Declare the **`host:port`** as an **insecure registry** so the **Docker daemon** allows HTTP.

**Docker Desktop:** **Settings → Docker Engine** → merge **`insecure-registries`** into your JSON (keep other keys):

```json
"insecure-registries": ["192.168.1.100:5000"]
```

**Apply & restart**. Match the address to **`REGISTRY`** if yours differs.

**Linux:** put the same entry in **`/etc/docker/daemon.json`**, then **`sudo systemctl restart docker`**.

**2 — Buildx still failing after that:** `docker buildx build --push` sends the image through **BuildKit**, which often **does not** use the engine’s **`insecure-registries`** and may still try HTTPS to your LAN registry.

The **[`bot/deploy.sh`](bot/deploy.sh)** script avoids that for the **default single platform** (`linux/arm64`): it builds with **`--load`** into the local Docker store, then runs **`docker push`**, so the **daemon** performs the push and your insecure registry setting applies.

If you use **multi-arch** (`PLATFORM=linux/amd64,linux/arm64`), the script uses **`buildx --push`**; if that hits the same HTTPS error, either push **arm64-only** with the default script, or configure **BuildKit** for HTTP to your registry (see [Docker BuildKit: registry configuration](https://docs.docker.com/build/buildkit/configure/#registry-configuration)).

On the Pi, pull and run. Variables already set in the baked **`.env`** apply; you can override any of them at runtime (`docker run -e …` overrides the file):

```bash
docker pull 192.168.1.100:5000/lifesoundtrack-bot:latest
docker run -d --name lifesoundtrack-bot --restart unless-stopped \
  192.168.1.100:5000/lifesoundtrack-bot:latest
```

Replace **`192.168.1.100`** with your Pi’s LAN address if it differs.
