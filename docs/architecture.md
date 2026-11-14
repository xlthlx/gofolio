# goFolio CMS – Architecture

goFolio is a self-hosted content management system built from three moving
parts: a chi router, a single SQLite file, and a set of HTML templates. The
public side answers with JSON, the admin side answers with server-rendered HTML
that HTMX progressively enhances. There is no message queue, no cache, no
background worker and no second datastore to reason about.

## Startup sequence

Everything happens in `main` in `cmd/server/main.go`, in a fixed order:

1. `LoadConfig` reads the environment (`cmd/server/config.go`).
2. `store.Open` opens the SQLite file and pins the pool to one connection.
3. `store.Migrate` applies the idempotent `CREATE TABLE IF NOT EXISTS` schema.
4. `service.InitUsersTable` is called, which is a compatibility alias for
   `store.Migrate` and adds nothing on top of it.
5. `service.CreateDefaultAdmin` inserts the `admin` account if no administrator
   row exists yet.
6. If the first argument is `-seed-only`, the process stops here and exits.
   This is how `make seed` prepares a database without starting a server.
7. `web.FindAdminThemeDir` locates the admin theme directory and
   `web.MustRenderer` parses every template group once. A template that fails
   to parse panics, so a broken theme never reaches production half-working.
8. `middleware.NewSessions` builds the cookie store from `AUTH_SECRET`, or from
   a random secret when the variable is unset.
9. `core.New` assembles the router and returns an `http.Handler`.
10. `http.Server` runs with a 10 second read timeout, a 30 second write timeout
    and a 60 second idle timeout, and shuts down gracefully on `SIGINT` or
    `SIGTERM` with a 5 second grace period.

## Layers

The layering is thin and direct, and each package has one job.

| Package | Responsibility |
|---|---|
| `cmd/server` | Process entry point and configuration loading |
| `internal/core` | The route table, built in `router.go` |
| `internal/handler/web` | HTTP handlers, template rendering, slug generation |
| `internal/handler/middleware` | Session cookie store and the auth middleware |
| `internal/service` | User and authentication rules |
| `internal/store` | SQLite access, one store type per table group |
| `internal/model` | Plain structs with JSON tags |
| `pkg/util` | bcrypt password hashing |

Content does not travel through `internal/service`. Posts and pages are read and
written by stores that handlers construct inline, which keeps the number of
indirection layers down. `internal/service` exists for the parts that carry real
rules, namely account creation, login and password changes.

## A request end to end

Taking `GET /admin/posts/` as the example:

1. The chi middleware chain runs first: request logging, panic recovery, real
   client IP resolution, and the trailing slash redirect.
2. The path matches the `/admin/` route group, so `middleware.RequireAuth`
   checks the session cookie. Without a valid session the request is redirected
   to `/admin/login/` with status 303 and nothing else runs.
3. `Admin.PostsPage` calls `h.postsData()`, which builds
   `store.PostStore{DB: h.DB}` and calls `ListAll()`.
4. The rows are wrapped in `PostsData` and handed to
   `Renderer.PostsPage`, which executes the `posts.html` template that was
   parsed at startup.
5. The handler has already set `Content-Type: text/html; charset=utf-8`, so the
   browser renders the page directly, with the table rows present in the first
   byte of the response.

An HTMX request follows the same path. The only difference is the header
`HX-Request: true`, which tells a handler to return a fragment instead of a
whole page, or to answer with `HX-Redirect` rather than a 303.

## Dependency wiring

`core.New(db, tpl, sess)` receives its database, templates and session store as
arguments. Handlers hold what they need as struct fields (`Admin` holds `DB`,
`Tpl` and `Sess`; `Public` holds `DB`) and use value receivers. Sessions used to
live in a package-level variable, and that was removed on purpose so tests can
inject their own secret and so no request can quietly depend on global state.

There is no dependency injection container and no repository interface for
stores. When a handler needs a store, it builds one from `h.DB`. `Admin` also
exposes the small helpers `h.posts()`, `h.pages()` and `h.settings()` for this,
which keeps store construction out of `if` statement headers where Go's
composite literal parsing would be ambiguous.

## Deliberate omissions

The following are absent by design rather than by accident. There is no public
HTML rendering, so the homepage is a JSON feed and the public theme directory
only serves assets. There is no migration version table, so schema changes have
to be written to be safe on a database that already exists. There is no plugin
or theme loading mechanism beyond the `THEMES_DIR` override. There is no media
upload endpoint, despite the placeholder packages `pkg/database`, `pkg/logger`
and `api/v1` that exist as empty `.gitkeep` files.

## Related documents

Read [routing.md](routing.md) for the full route table, [templates.md](templates.md)
for how rendering and HTMX fit together, [database.md](database.md) for the
schema and store methods, and [authentication.md](authentication.md) for sessions
and login. [folder-structure.md](folder-structure.md) lists where each file
lives.
