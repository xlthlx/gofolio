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

The public site is HTML, rendered by the handlers in `internal/handler/web/site.go`.

| Method | Path | Behaviour |
|---|---|---|
| `GET` | `/` | Homepage listing published posts, newest first |
| `GET` | `/posts/{slug}/` | One published post; a draft or unknown slug is a themed 404 |
| `GET` | `/pages/{slug}/` | One published page; a draft or unknown slug is a themed 404 |
| `GET` | `/static/*` | Files from `web/themes/public/default/` |
| `GET` | `/healthz/` | Liveness plus a database ping: `200 {"status":"ok"}` or `503` |

Every public page reads the site title and tagline from the settings table,
falling back to `goFolio` and `A simple Go CMS` when nothing is stored, and
renders its navigation according to the session: a login link when signed out, and
admin and logout links when signed in. Pages are rendered from scratch on every
request, so an edit is visible immediately.

The same content is also published as JSON, which is what the machine-readable
clients use and what the tests exercise.

| Method | Path | Behaviour |
|---|---|---|
| `GET` | `/api/posts/` | Published posts, newest first, wrapped as `{"posts":[...]}` |
| `GET` | `/api/posts/{slug}/` | One published post as JSON; a draft or unknown slug is a `404` |
| `GET` | `/api/pages/{slug}/` | One published page as JSON; a draft or unknown slug is a `404` |

These moved under `/api/` when the site became HTML: a URL now has one
representation, rather than switching on the `Accept` header.

## Admin pages and fragments

These routes are inside the `/admin/` group and therefore require a session.

| Method | Path | Behaviour |
|---|---|---|
| `GET` | `/admin/` | Dashboard, same handler as `/admin/dashboard/` |
| `GET` | `/admin/dashboard/` | Dashboard with statistics and recent posts computed server side |
| `GET` | `/admin/dashboard/recent/` | `recent_rows` fragment, polled by the dashboard every 30 seconds |
| `GET` | `/admin/posts/` | Posts page, table rows rendered inline |
| `GET` | `/admin/posts/table/` | `posts_rows` fragment only |
| `GET` | `/admin/posts/new/` | Post editor, rendered empty |
| `GET` | `/admin/posts/{id}/edit/` | Post editor rendered with the stored post; an unknown id is a 404 |
| `POST` | `/admin/posts/save/` | Saves the post editor form, creating or updating on the id it carries |
| `GET` | `/admin/pages/` | Pages page, table rows rendered inline |
| `GET` | `/admin/pages/table/` | `pages_rows` fragment only |
| `GET` | `/admin/pages/new/` | Page editor, rendered empty |
| `GET` | `/admin/pages/{id}/edit/` | Page editor rendered with the stored page |
| `POST` | `/admin/pages/save/` | Saves the page editor form, creating or updating |
| `GET` | `/admin/settings/` | Settings page: site settings with stored values, plus the password form |
| `POST` | `/admin/settings/password/` | Password change, minimum eight characters |
| `POST` | `/admin/settings/site/` | Saves the whitelisted site settings; answers with the re-rendered section for HTMX and with a `303` to `/admin/settings/` otherwise |
| `GET` | `/admin/static/*` | Files from `web/themes/admin/default/` |

The table fragment endpoints exist as the pieces the tables are built from, and
the tests exercise them, but the posts and pages pages currently render the same
`{{template "posts_rows" .}}` include inline rather than fetching it. Only the
dashboard polls a fragment.

The editor routes come in pairs. The two `GET` routes share a handler, which
renders the form with the stored values already in place, and the id in the path
is what tells them apart. Both `POST /save/` endpoints take the whole form,
including a hidden id, and treat a request without one as a new record, so one
handler covers creating and updating. TinyMCE owns the rich text field, so the
browser sends the editor's HTML back in the same form; see
[templates.md](templates.md).

## Admin JSON API

Both resources live under `/admin/`, so the same session requirement applies.
These endpoints are the JSON counterpart of the editors, used by the delete
buttons in the tables and available to any client that prefers JSON to forms.

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
| `PUT` | `/admin/api/pages/{id}/` | Replaces title, slug, content and status; returns `{"status":"updated"}` |
| `DELETE` | `/admin/api/pages/{id}/` | Same HTMX and API split as posts |

Creation is forgiving in two ways: an empty `slug` is derived from the title, and
an empty `status` becomes `draft`. A missing title is rejected with
`400 Title is required`. Both rules apply to pages as well as posts, since the
editor forms and the JSON API go through the same store methods, and the status
default also applies on update, so a row never ends up with an empty status.

`PUT` is used where `PATCH` might be expected. The request body is decoded into a
fresh model and all editable fields are written, so a partial body clears the
fields it omits. Posts and pages both set `updated_at` on update.

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

Saves follow the same split. The editor forms post over HTMX, so a successful
save answers `200` with `HX-Redirect: /admin/posts/` and an empty body, while a
failure answers the status that describes it with an error fragment the form
swaps into place. Without HTMX the same handler redirects with `303` on success
and re-renders the editor with the error on failure, so nothing depends on
JavaScript being available.

JSON errors are written with `http.Error`, which produces a `text/plain`
response, so an error body is JSON shaped only when it was written by
`writeJSON`. The health endpoint is the clearest example: the healthy answer is
`application/json`, while the unhealthy answer is plain text.
