# goFolio CMS – Development

This document covers the working loop: what to install, which commands exist,
how configuration is read, and where new code belongs.

## Requirements

Go 1.26 or newer is required, as pinned in `go.mod`. SQLite support is compiled
in through `mattn/go-sqlite3`, so building and testing need a C toolchain and
`CGO_ENABLED=1`. On Alpine that means `gcc` and `musl-dev`; on other systems a
normal `gcc` or `clang` install is enough. Docker is only needed if you want to
run the container image.

## Commands

The `Makefile` wraps the common tasks. `PORT` and `DOMAIN` can be overridden on
the command line and default to `8080` and `localhost`.

| Command | What it does |
|---|---|
| `make run` | `go run ./cmd/server` with `PORT` and `DOMAIN` |
| `make dev` | Same as `run`, after `go mod tidy` |
| `make build` | Writes the `./gofolio` binary via `go build -o gofolio ./cmd/server` |
| `make test` | `go test -v ./...` |
| `make seed` | `go run ./cmd/server -seed-only`, then exits |
| `make deps` | `go mod tidy` and `go mod verify` |
| `make clean` | Removes the `gofolio` binary and every `*.db` file in the repo root |
| `make docker-dev` | `docker-compose up --build -d` |
| `make docker-down` | `docker-compose down` |

`make clean` deletes all SQLite files in the repository root, which means the
local database and its content. Use it only when you want a fresh start.

## Running locally

`make run` starts the server on port 8080, creating the database file, applying
the schema and creating the default administrator `admin` with the password
`password` on first start. The startup log repeats that reminder.

`make seed` prepares a database without running a server, which is useful in
provisioning scripts. It creates the file, applies the schema, ensures the
administrator exists, prints `Database initialized and default admin ensured.`
and exits. The `-seed-only` flag has to be the first argument, because `main`
inspects `os.Args[1]` directly.

## Configuration

Configuration comes from environment variables, all of which have defaults
suitable for local work.

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | Port the HTTP server listens on |
| `DOMAIN` | `localhost` | Only used in the startup banner |
| `DATABASE_PATH` | `./gofolio.db` | Path to the SQLite file |
| `AUTH_SECRET` | random per start | Session cookie signing key |
| `THEMES_DIR` | auto-discovered | Overrides template lookup; expects `<THEMES_DIR>/admin/default` |

If `AUTH_SECRET` is unset, a random secret is generated at each start, which
invalidates every existing session on restart. Set it explicitly in any
environment where sessions should survive a restart.

`THEMES_DIR` exists so templates can live outside the repository. Without it,
`FindAdminThemeDir` walks up from the current working directory until it finds
`web/themes/admin/default/posts.html`, which is why tests launched from any
package directory still locate the templates.

## Dependencies and vendoring

Dependencies are vendored into `vendor/`, and that directory is committed, so
`go build`, `go test` and `go vet` run in vendor mode automatically. After
changing a dependency, run `go mod tidy` followed by `go mod vendor` to keep
`vendor/modules.txt` in sync, otherwise Go fails with an inconsistent vendoring
error. The Docker build does not copy `vendor/` and resolves modules through
`go mod download` instead.

The direct dependencies are small: `go-chi/chi/v5` for routing,
`gorilla/sessions` for cookies, `mattn/go-sqlite3` for the database, and
`golang.org/x/crypto` for bcrypt.

## Where new code belongs

Adding an admin page touches three places, and the order is usually templates
first: create the template in `web/themes/admin/default/`, add it to the right
`template.ParseFiles` group in `NewRenderer` in `internal/handler/web/render.go`
with a matching method on `Renderer`, then register the route in
`internal/core/router.go`, remembering the trailing slash.

Adding a route means editing `internal/core/router.go` only for registration,
with the handler itself living in `internal/handler/web`. Routes inside the
`/admin/` group are protected automatically by `middleware.RequireAuth`; routes
that must stay public (`/admin/login/`, `/admin/static/*`, `/api/auth/*`) are
registered outside that group.

Adding a store method means editing `internal/store`, adding a test next to it,
and calling it from a handler. Adding a column means editing the migration blob
in `internal/store/db.go` in a way that is safe to re-run, which
[database.md](database.md) explains in more detail.

Anything that links between pages should keep the trailing slash on internal
URLs. A missing slash still works, but only after an extra redirect round trip.

## Conventions

Code follows the surrounding style closely: focused files, one concern per file,
and comments that explain why something is done rather than restating the code.
Every Go change is expected to come with tests in a file next to the code under
test, as described in [testing.md](testing.md). `go vet ./...` is clean today and
there is no configured linter or formatter beyond `gofmt`.

## Troubleshooting

A `database is locked` error means more than one connection is writing. The pool
is intentionally limited to one connection in `store.Open`, so this should not
happen unless something opens the database separately from that function.

A panic at startup mentioning template parsing means `MustRenderer` could not
parse a template group, usually because a `{{template "name"}}` include refers to
a file that is not part of the same `ParseFiles` group. See
[templates.md](templates.md).

Being sent back to `/admin/login/` even after signing in means the session cookie
cannot be read on the next request, which happens when `AUTH_SECRET` changes
between them. That is the normal case when the variable is unset, because a new
random secret is generated on every start. Setting a fixed `AUTH_SECRET` resolves
it.
