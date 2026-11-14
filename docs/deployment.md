# goFolio CMS – Deployment

goFolio deploys as a single binary and a themes directory, with one SQLite file
for state. This document covers building it, running it in Docker, and what to
check before putting it on a public address.

## Building the binary

`make build` runs `go build -o gofolio ./cmd/server` after resolving modules. The
result is one executable that still needs the `web/` directory beside it, because
templates and assets are read from disk rather than embedded. Building requires a
C toolchain, since the SQLite driver uses CGO. The binary links against the
system C library, so deploying it on a minimal Linux image usually means
installing `libc6-compat` or `libc6`.

## The container image

The `Dockerfile` is a two stage build. The builder stage starts from
`golang:1.26-alpine`, downloads modules with `go mod download`, installs `gcc`
and `musl-dev`, copies the source directories, and builds with
`CGO_ENABLED=1 GOOS=linux` plus `-ldflags="-s -w"` and `-trimpath` to a binary at
`/bin/gofolio`. Module resolution happens over the network here, because the
build context does not include the vendored dependencies.

The runtime stage is `alpine:latest` with `ca-certificates`, `sqlite` and
`libc6-compat` installed. It creates a `gofolio` group and user, runs as that
non-root user, sets `WORKDIR /app`, copies the binary and `web/`, creates
`/app/data`, and exports `PORT=8080` with
`DATABASE_PATH=/app/data/gofolio.db`. The image exposes port 8080, defines a
health check that requests `/healthz/` every 30 seconds with
`wget`, and uses the binary as its entry point, so any extra arguments are passed
straight to it.

## Docker Compose

`docker-compose.yml` builds the local directory and publishes
`${PORT:-8080}` on port 8080 inside the container. Three details in that file are
worth knowing.

The project root is mounted at `/app/data`, so the container's database file is
the `gofolio.db` in your working copy. The side effect is that the container can
see the whole repository, including its source, and that the database lands in
the repository root rather than in a Docker volume.

The service sets `user: "${UID:-1000}:${GID:-1000}"` so it runs as the invoking
user. The image's own user has uid 100 and cannot write a bind-mounted file owned
by uid 1000, which would leave SQLite unable to write its journal.

`DATABASE_PATH` is set to `/app/data/gofolio.db`, which matches the mount, and
the service restarts unless stopped. The image health check applies here as well,
so `docker compose ps` reports the container as healthy only once the health
endpoint answers.

`make docker-dev` runs `docker-compose up --build -d`, `make docker-down` stops
and removes the container. Logs go to standard output through the chi request
logger and the `log` package, so `docker compose logs -f` is the way to watch
them.

## Configuration in production

| Variable | Recommendation |
|---|---|
| `AUTH_SECRET` | Set it to a long random value; without it sessions die on every restart |
| `DATABASE_PATH` | Point it at a writable, backed-up location |
| `DOMAIN` | Cosmetic, only used in the startup banner |
| `THEMES_DIR` | Set it when templates live outside the working directory |
| `PORT` | Defaults to 8080 |

## Operational notes

The health endpoint pings the database as well as answering, so a container that
reports unhealthy usually means SQLite is not reachable or not writable rather
than that the process is gone. Graceful shutdown is handled: `SIGTERM` and
`SIGINT` start a shutdown with a five second grace period, which fits inside the
default ten second stop timeout Docker allows.

There is no built-in TLS, and the session cookie is issued without the `Secure`
flag, so the usual arrangement is a reverse proxy that terminates TLS and
forwards to port 8080. Two behaviours to keep in mind when configuring that
proxy: a `GET` without a trailing slash answers with a permanent redirect, and
static assets are served with `http.ServeFile`, which sets no cache headers, so
caching decisions belong to the proxy.

Backups are file copies. Stop the container, copy `gofolio.db`, and start it
again, or use the `sqlite3` `.backup` command against a running database. There
is no other state to preserve, since uploaded media does not exist yet.

To create the database and the default administrator before the first request,
run the binary once with `-seed-only`, for example with
`docker compose run --rm gofolio -seed-only`, which prepares admin credentials
without serving any traffic. Whichever route you take, log in as `admin` and
change the password immediately.
