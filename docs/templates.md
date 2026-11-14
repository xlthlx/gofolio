# goFolio CMS – Templates and Rendering

The admin panel is server-rendered HTML, progressively enhanced with HTMX, with
two deliberate exceptions where JavaScript owns the page state. This document
explains where templates live, how they are parsed, which fragments exist, and
what to touch when adding a page.

## Themes and assets

Templates and their assets live under `web/themes`, split into an admin theme at
`web/themes/admin/default` and a public theme at `web/themes/public/default`.
Static files are served straight from those directories with `http.ServeFile`,
so `/admin/static/*` maps to the admin theme and `/static/*` maps to the public
theme. Nothing is embedded in the binary, which is why the Docker image copies
`web/` next to the binary and why tests walk up the tree to find the repository.

The assets that exist today are a stylesheet, the vendored HTMX build, the
vendored TinyMCE distribution under `assets/js/tinymce/`, and empty
`assets/img` directories. The public theme contains a stylesheet and
`index.html`, but only its assets directory is reachable: `index.html` is not
routed, and it links to `/static/styles.css`, a path that does not correspond to
a file in the theme. Treat that file as a leftover rather than a working page.

Styling leans on element selectors, with classes reserved for buttons and a few
layout helpers. Pages are expected to stay semantic and to avoid icons.

## Template groups

`NewRenderer` in `internal/handler/web/render.go` parses the templates once at
startup into five independent groups. A `{{template "name"}}` include can only
resolve to a file parsed into the same group, so groups are the unit to keep in
mind when editing.

| Group | Files parsed | Template names inside |
|---|---|---|
| posts | `posts.html`, `posts_table.html` | `posts.html`, `posts_rows` |
| pages | `pages.html`, `pages_table.html` | `pages.html`, `pages_rows` |
| settings | `settings.html`, `site_settings.html` | `settings.html`, `site_settings` |
| dashboard | `dashboard.html`, `dashboard_recent.html` | `dashboard.html`, `recent_rows` |
| login | `login.html` | `login_page`, `login_error` |
| post editor | `editor.html` | `editor_page`, `editor_error` |
| page editor | `page_editor.html` | `page_editor_page`, `page_editor_error` |

`Renderer` exposes one method per thing it renders, such as `PostsPage`,
`PostsRows`, `DashboardPage` and `LoginError`, and those are what handlers call.
`MustRenderer` parses the same groups but panics on failure, and `main` uses it
so a theme that cannot be parsed stops the server instead of producing broken
pages. `FindAdminThemeDir` resolves the directory, honouring `THEMES_DIR` first
and otherwise walking up from the working directory until it finds
`web/themes/admin/default/posts.html`.

## Fragments and HTMX

Three fragments are rendered by dedicated routes: `recent_rows` for the
dashboard, and `posts_rows` and `pages_rows` for the two tables. The posts and
pages pages include those same row templates inline at render time, so the page
arrives complete, while the fragment routes remain available for a targeted
refresh. The dashboard is the one place that polls: the container element
carries `hx-get="/admin/dashboard/recent/"` with `hx-trigger="every 30s"` and
`hx-swap="innerHTML"`.

Row delete buttons use `hx-delete` against `/admin/api/posts/{id}/` or
`/admin/api/pages/{id}/` with `hx-confirm` for the prompt and
`hx-target="closest tr"` plus `hx-swap="outerHTML"` so the row disappears. Those
responses are empty bodies on purpose, which is why the handlers answer `200`
for HTMX rather than `204`.

Three forms swap their result into the page in place: the login form posts to
`/admin/login/` and targets `#login-error`, the password form posts to
`/admin/settings/password/` and targets `#password-result`, and the two editors
post to `/admin/posts/save/` and `/admin/pages/save/` with their error fragment
targeted at an empty result container. The password handler writes a small
paragraph of HTML with an inline colour, which is the one place where styling is
emitted from Go rather than from the stylesheet; the error fragments do the same
to stay consistent with it.

Every one of these flows also works as a plain form submission. The forms carry
a conventional `method` and `action` alongside the `hx-` attributes, and the
handlers answer accordingly, so the admin remains usable with JavaScript
disabled. HTMX is always detected the same way, by checking whether the
`HX-Request` header equals `true`.

## The two editors

The two editors follow the same pattern as every other admin page, with one
complication: TinyMCE owns the rich text field, and it keeps its own copy of the
document. The forms therefore arrive fully rendered by the server, including the
stored title, slug, content and status, and each one posts its whole self back to
`/admin/posts/save/` or `/admin/pages/save/`. A hidden `id` field decides whether
the save creates or updates, so one handler serves both modes.

The only JavaScript left in each editor is the TinyMCE initialisation plus a
submit listener that calls `tinymce.triggerSave()`, which copies the editor's
HTML back into the textarea before the form is serialised. That listener is
registered while the page is parsed, and `htmx.min.js` is loaded with `defer`, so
the textarea is current before HTMX collects the form values. Loading needs no
request at all: the textarea already holds the content when TinyMCE initialises
over it.

Because the server derives the slug, and because the form always submits the
stored slug when one exists, the editors carry no client-side slug preview. A
title typed into an empty slug field is turned into a slug by `Slugify` in
`admin.go` when the save arrives.

## Settings page detail

`GET /admin/settings/` renders `settings.html` with no data, and
`GET /admin/settings/site/` renders the same template with the stored values.
The `site_settings` include tolerates a missing map, falling back to the default
site title and tagline, which is why the settings page renders fine on a database
with no settings rows. The form posts only the whitelisted keys `site_title` and
`tagline`, empty values are skipped rather than saved, and the handler re-renders
the whole settings page afterwards so the saved values are visible immediately.

## Adding an admin page

Three places have to change together. First, add the template file, and make sure
any `{{template}}` include it uses is parsed into the same group. Second, add the
file to the matching `template.ParseFiles` call in `NewRenderer` and add a
`Renderer` method that executes the right template name. Third, register the
route in `internal/core/router.go` with a trailing slash. If the page needs the
admin shell, the existing pages are the pattern to copy: the same header, nav and
stylesheet link appear at the top of `posts.html`, `pages.html`, `settings.html`,
`dashboard.html` and both editors.
