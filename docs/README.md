# goFolio CMS – Documentation

These documents describe goFolio CMS as it is built today: a chi router, one
SQLite file, a JSON API for the public site, and a server-rendered admin panel
progressively enhanced with HTMX. Each document stands on its own, but they refer
to each other so a question that starts in one place can be followed to the
detail it needs.

## The documents

| Document | What it covers |
|---|---|
| [architecture.md](architecture.md) | Startup sequence, the packages and what each one owns, how a request travels from router to template, and what is deliberately absent |
| [development.md](development.md) | Requirements, every `make` target, environment variables, vendoring, where new code goes, and common startup problems |
| [routing.md](routing.md) | The complete route table, the middleware stack, the trailing slash rule, status code conventions, and the JSON shapes the editors depend on |
| [templates.md](templates.md) | Theme and asset layout, the five template groups, which fragments exist, how HTMX is wired, and what to touch when adding an admin page |
| [database.md](database.md) | The schema, how migrations work without a version table, what each store does, and the side effects to expect from writes |
| [authentication.md](authentication.md) | Password hashing, the default administrator, session cookies, both login flows, password changes, and the known security gaps |
| [testing.md](testing.md) | Test layout per package, the helper to reuse in each one, what the integration suite verifies, and what is not covered |
| [deployment.md](deployment.md) | Building the binary, the Docker image and Compose setup, production configuration, health checks, backups and TLS placement |
| [folder-structure.md](folder-structure.md) | The file tree with a note on what each file and directory holds |

## Where to start

If you want to run the project, read [development.md](development.md) for the
local loop and [deployment.md](deployment.md) for containers and production. Both
cover the default `admin` / `password` account, which exists so a fresh database
can be logged into immediately.

If you are extending it, read [architecture.md](architecture.md) first to see how
the layers fit, then the document for the area you are touching:
[routing.md](routing.md) for endpoints, [templates.md](templates.md) for pages and
fragments, [database.md](database.md) for anything touching storage, and
[testing.md](testing.md) for how the change is expected to be verified.

If you are reviewing it, [authentication.md](authentication.md) lists the security
properties and the known gaps, and [database.md](database.md) explains the
migration constraints that shape schema changes.

## A short orientation

The public side answers with JSON: the homepage is a feed of published posts, and
posts and pages are fetchable by slug. There is no public HTML theme in use, so
the public theme directory currently only serves assets. The admin side is the
part with a user interface, reachable at `/admin/login/` and protected by a
session cookie. Admin pages arrive as complete server-rendered HTML, and the
interactive parts, which are table row deletion, dashboard refresh, and login and
password feedback, are HTMX requests that receive fragments. The two content
editors are the exception and talk JSON to the admin API, because TinyMCE owns the
document state. The root [README](../README.md) covers the project itself at a
higher level, including its license and credits.
