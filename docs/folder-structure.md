# goFolio CMS – Folder Structure

The tree below is the repository as it stands, with a note on what each file
holds.

```
gofolio/
├── cmd/
│   └── server/
│       ├── config.go             # Environment variable loading
│       └── main.go               # Startup sequence and graceful shutdown
│
├── docs/
│   ├── README.md                 # Index of the documentation
│   ├── architecture.md           # Layers and request lifecycle
│   ├── authentication.md         # Sessions, login and security posture
│   ├── database.md               # Schema, migrations and stores
│   ├── deployment.md             # Binary, Docker image and Compose
│   ├── development.md            # Commands, configuration and conventions
│   ├── folder-structure.md       # This file
│   ├── routing.md                # Route table and HTTP conventions
│   ├── templates.md              # Rendering, fragments and HTMX
│   └── testing.md                # Test layout and helpers
│
├── internal/
│   ├── core/
│   │   └── router.go             # Route table and middleware chain
│   │
│   ├── handler/
│   │   ├── middleware/
│   │   │   ├── auth.go           # RequireAuth middleware
│   │   │   └── session.go        # Cookie session store
│   │   │
│   │   └── web/
│   │       ├── admin.go          # Post and page handlers, slug generation
│   │       ├── auth.go           # /api/auth endpoints
│   │       ├── common.go         # JSON, HTMX and id helpers, static files
│   │       ├── dashboard.go      # Dashboard page and recent posts fragment
│   │       ├── editor.go         # Post editor page and form save
│   │       ├── health.go         # Liveness endpoint with database ping
│   │       ├── login.go          # Admin login page and form post
│   │       ├── page_editor.go    # Page editor page and form save
│   │       ├── public.go         # Public JSON API
│   │       ├── render.go         # Template parsing and the Renderer
│   │       ├── settings.go       # Settings page and password change
│   │       └── settings_site.go  # Site settings form
│   │
│   ├── model/
│   │   ├── content.go            # Post and Page structs
│   │   └── user.go               # User struct
│   │
│   ├── service/
│   │   └── user.go               # User creation, login, password change
│   │
│   └── store/
│       ├── content_store.go      # PostStore and PageStore
│       ├── db.go                 # Connection handling and schema
│       └── settings_store.go     # SettingStore
│
├── pkg/
│   ├── database/                 # Placeholder, empty
│   ├── logger/                   # Placeholder, empty
│   └── util/
│       └── hash.go               # bcrypt password hashing
│
├── web/
│   └── themes/
│       ├── admin/default/        # Admin templates and static assets
│       └── public/default/       # Public assets; index.html is not routed
│
├── test/
│   ├── fixtures/                 # Placeholder, empty
│   └── integration/
│       └── api_test.go           # End-to-end tests through the real router
│
├── api/v1/                       # Placeholder, empty
├── assets/                       # Project logo and banner
├── vendor/                       # Vendored dependencies, committed
│
├── docker-compose.yml
├── Dockerfile
├── LICENSE
├── Makefile
├── README.md
├── go.mod
├── go.sum
```

Test files are not listed individually because they follow the source files
exactly: every Go file in `internal/` that carries logic has a matching
`_test.go` beside it, and the handler package keeps its shared test scaffolding
in `internal/handler/web/common_test.go`.

## Placeholders and leftovers

Several directories exist only to reserve a place: `pkg/database`, `pkg/logger`,
`api/v1`, `scripts` and `test/fixtures` contain nothing but `.gitkeep` files.
Earlier drafts of this document also named an engine in `internal/core`, a REST
handler file, media handling across the model, service and store layers, an
OpenAPI specification and a migration script. None of those exist, and this tree
deliberately omits them so it can be trusted as a map.

## Key design principles

The layout follows the conventional Go project shape, with application code under
`internal`, one entry point under `cmd`, and reusable helpers under `pkg`. Files
are small and each one covers a single responsibility, so a reader looking for
behaviour can usually guess the file from the feature.

Testability is built into the shape rather than added on top. Packages that need
internals use internal tests, the handler package is tested through its exported
API, and the integration suite boots the real router so middleware and status
codes are covered too. Nothing depends on package-global state, which is why
sessions and templates are injected into `core.New` instead of being created
where they are used.

Growth is expected along the existing seams. A new resource gains a store in
`internal/store`, a handler in `internal/handler/web`, a template group and a set
of routes, in that order of dependency. Media support, when it arrives, would
extend the same three layers rather than introduce a new pattern.

## Naming conventions

Packages are lowercase and descriptive, as in `store`, `service` and
`middleware`. Files are named after their primary responsibility, so
`settings_site.go` holds the site settings handler and `content_store.go` holds
the content stores. Tests mirror their source file exactly, which is what keeps
the handler package's one-file-per-source layout readable. Templates are named
after the page they render, and fragment templates take a `_table` or `_recent`
suffix.
