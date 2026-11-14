# goFolio CMS – Routing and HTTP

Every route is registered in one place, `New` in `internal/core/router.go`.
Handlers are methods on structs in `internal/handler/web`. This document lists
the route table, the middleware that wraps it, and the status code conventions
the handlers follow.

## Middleware

Three chi middleware run for every request: `Logger`, `Recoverer` and `RealIP`.
A fourth, registered inline, redirects any `GET` whose path does not end in a
slash to the same path with a slash, preserving the query string and answering
with a permanent redirect. Only `GET` is treated this way. A `POST` to a path
without the trailing slash does not match any route and is not redirected, so
internal links and form actions always need the trailing slash.

Authentication is not global. The `/admin/` route group adds
`middleware.RequireAuth`, which checks the session and redirects unauthenticated
requests to `/admin/login/` with status 303. Login, admin static assets and the
`/api/auth/` endpoints sit outside the group on purpose, because they have to be
reachable before a session exists.

## Public routes

| Method | Path | Behaviour |
|---|---|---|
| `GET` | `/` | JSON feed of published posts, newest first, wrapped as `{"posts":[...]}` |
| `GET` | `/healthz/` | Liveness plus a database ping: `200 {"status":"ok"}` or `503` |
| `GET` | `/posts/{slug}/` | One published post as JSON; a draft or unknown slug is a 404 |
| `GET` | `/pages/{slug}/` | One page as JSON regardless of status; unknown slug is a 404 |
| `GET` | `/static/*` | Files from `web/themes/public/default/` |

The homepage is the JSON feed. There is no HTML homepage route, and
`web/themes/public/default/index.html` is not served by anything.

## Admin pages and fragments

These routes are inside the `/admin/` group and therefore require a session.

| Method | Path | Behaviour |
|---|---|---|
| `GET` | `/admin/` | Dashboard, same handler as `/admin/dashboard/` |
| `GET` | `/admin/dashboard/` | Dashboard with statistics and recent posts computed server side |
| `GET` | `/admin/dashboard/recent/` | `recent_rows` fragment, polled by the dashboard every 30 seconds |
| `GET` | `/admin/posts/` | Posts page, table rows rendered inline |
| `GET` | `/admin/posts/table/` | `posts_rows` fragment only |
| `GET` | `/admin/posts/new/` | Post editor, served as the static file `editor.html` |
| `GET` | `/admin/posts/{id}/edit/` | Same static editor file; the id comes from the URL |
| `GET` | `/admin/pages/` | Pages page, table rows rendered inline |
| `GET` | `/admin/pages/table/` | `pages_rows` fragment only |
| `GET` | `/admin/pages/new/` | Page editor, served as the static file `page_editor.html` |
| `GET` | `/admin/pages/{id}/edit/` | Same static page editor file |
| `GET` | `/admin/settings/` | Settings page: site settings plus the password form |
| `POST` | `/admin/settings/password/` | Password change, minimum eight characters |
| `GET` | `/admin/settings/site/` | Site settings form filled with stored values |
| `POST` | `/admin/settings/site/` | Saves whitelisted keys, then re-renders the page |
| `GET` | `/admin/static/*` | Files from `web/themes/admin/default/` |

The table fragment endpoints exist as the pieces the tables are built from, and
the tests exercise them, but the posts and pages pages currently render the same
`{{template "posts_rows" .}}` include inline rather than fetching it. Only the
dashboard polls a fragment.

The editor routes are worth a note: they serve `editor.html` and
`page_editor.html` as plain files through `Admin.View`, without executing them
as templates, because TinyMCE owns the content state and the page loads its data
through the JSON API. See [templates.md](templates.md).

## Admin JSON API

Both resources live under `/admin/`, so the same session requirement applies.
These endpoints are what the editors talk to.

| Method | Path | Behaviour |
|---|---|---|
| `GET` | `/admin/api/posts/` | All posts, drafts included, as a JSON array |
| `GET` | `/admin/api/posts/{id}/` | One post as JSON; any lookup failure is a 404 |
| `POST` | `/admin/api/posts/` | Creates a post and returns `201` with the stored post |
| `PUT` | `/admin/api/posts/{id}/` | Replaces title, slug, content and status; returns `{"status":"updated","id":"7"}` |
| `DELETE` | `/admin/api/posts/{id}/` | `200` with an empty body for HTMX, otherwise `204` |
| `GET` | `/admin/api/pages/` | All pages as a JSON array |
| `GET` | `/admin/api/pages/{id}/` | One page as JSON |
| `POST` | `/admin/api/pages/` | Creates a page; returns `201 {"status":"created"}` |
| `PUT` | `/admin/api/pages/{id}/` | Replaces title, slug and content; returns `{"status":"updated"}` |
| `DELETE` | `/admin/api/pages/{id}/` | Same HTMX and API split as posts |

Creation is forgiving in two ways: an empty `slug` is derived from the title, and
an empty `status` becomes `draft`. A missing title is rejected with
`400 Title is required`. Note that pages carry a `status` field in their JSON,
but `PageStore` neither reads nor writes that column, so it always reports
`draft`; [database.md](database.md) explains why.

`PUT` is used where `PATCH` might be expected. The request body is decoded into a
fresh model and all editable fields are written, so a partial body clears the
fields it omits. On update, posts set `updated_at` while pages have no such
column.

Malformed JSON and unparsable ids both answer with `400`. Unexpected database
errors answer with `500` and the raw error text.

## Login and auth endpoints

| Method | Path | Behaviour |
|---|---|---|
| `GET` | `/admin/login/` | Login page; an already authenticated visitor is redirected to `/admin/` with 303 |
| `POST` | `/admin/login/` | Form login, described below |
| `POST` | `/api/auth/login/` | JSON login taking `{"username":"...","password":"..."}` |
| `GET` | `/api/auth/logout/` | Clears the session and returns `{"success":"logged out"}` |
| `GET` | `/api/auth/check/` | Returns `{"authenticated":true}` or `false` |

The form login behaves differently depending on whether HTMX sent the request.
Over HTMX, a failure returns status 401 with the `login_error` fragment so it can
be swapped into the page, and a success returns 200 with an `HX-Redirect` header
pointing at `/admin/`. Without HTMX, a failure is a 401 with the plain text
`Invalid credentials` and a success is a 303 redirect. The JSON endpoint answers
`200 {"success":"logged in"}` or `401 {"error":"Invalid credentials"}` and never
redirects, which is what the integration tests use.

Logout is a `GET`, so it can be triggered from a plain link, which is how the
admin templates do it. That also means it can be triggered by any page that
embeds a link to it.

## Status code conventions

Handlers translate `sql.ErrNoRows` into `404` with `errors.Is`, which keeps
missing rows distinct from real failures. Deletes answer `200` with an empty body
when `HX-Request: true` so the caller can drop the table row with
`hx-swap="outerHTML"`, and `204` for plain API clients. Everything that renders
HTML sets `Content-Type: text/html; charset=utf-8` explicitly before writing,
and JSON uses the `writeJSON` helper in `internal/handler/web/common.go`.

JSON errors are written with `http.Error`, which produces a `text/plain`
response, so an error body is JSON shaped only when it was written by
`writeJSON`. The health endpoint is the clearest example: the healthy answer is
`application/json`, while the unhealthy answer is plain text.
