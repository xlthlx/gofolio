# goFolio CMS – Templates and Rendering

The public site and the admin panel are both server-rendered HTML, progressively
enhanced with HTMX. The editors are the one place where JavaScript owns the page
state, because a rich text field needs it. This document explains where templates
live, how they are parsed, which fragments exist, and what to touch when adding a
page.

## Themes and assets

Templates and their assets live under `web/themes`, split into an admin theme at
`web/themes/admin/default` and a public theme at `web/themes/public/default`.
Static files are served straight from those directories with `http.ServeFile`,
so `/admin/static/*` maps to the admin theme and `/static/*` maps to the public
theme. Nothing is embedded in the binary, which is why the Docker image copies
`web/` next to the binary and why tests walk up the tree to find the repository.

The assets that exist today are a stylesheet in each theme, the vendored HTMX
build and a small `htmx-config.js` that adjusts how HTMX treats failed responses,
the vendored TinyMCE distribution under the admin theme's `assets/js/tinymce/`,
and empty `assets/img` directories. Both themes keep their own copy of the HTMX
files so a theme stays self-contained.

Styling leans on element selectors, with classes reserved for buttons and a few
layout helpers. Pages are expected to stay semantic and to avoid icons.

## Template groups

`NewRenderer` in `internal/handler/web/render.go` parses the templates once at
startup into one group per page family. A `{{template "name"}}` include can only
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
| public site | `layout.html`, `index.html`, `post.html`, `page.html`, `not_found.html` | `site_head`, `site_nav`, `index_page`, `post_page`, `page_page`, `not_found_page` |

`Renderer` exposes one method per thing it renders, such as `PostsPage`,
`PostsRows`, `DashboardPage` and `HomePage`, and those are what handlers call.
`MustRenderer` parses the same groups but panics on failure, and `main` uses it
so a theme that cannot be parsed stops the server instead of producing broken
pages. It takes both theme directories, which `FindAdminThemeDir` and
`FindPublicThemeDir` resolve, honouring `THEMES_DIR` first and otherwise walking
up from the working directory until they find `web/themes/admin/default/posts.html`
and `web/themes/public/default/index.html`.

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

Several forms swap their result into the page in place: the login form posts to
`/admin/login/` and targets `#login-error`, the password form posts to
`/admin/settings/password/` and targets `#password-result`, the site settings
form posts to `/admin/settings/site/` and replaces its own section, and the two
editors post to `/admin/posts/save/` and `/admin/pages/save/` with their error
fragment targeted at an empty result container. The password and error fragments
are small paragraphs of HTML with an inline colour, which is the one place where
styling is emitted from Go rather than from the stylesheet.

One limitation is worth knowing before adding another fragment. The vendored
HTMX 2.0.4 swaps only successful responses by default and treats 4xx and 5xx as
errors it does not swap, which would make every failure fragment invisible. The
admin templates therefore load `assets/js/htmx-config.js` right after the HTMX
build, where `defer` guarantees `htmx` exists, and that file sets
`responseHandling` so that 4xx and 5xx responses are swapped too. The handlers can
then keep answering with the status that describes the problem, and the message
they return is displayed in place. A test in `render_test.go` renders every page
template and fails if one of them stops loading the config.

Every one of these flows also works as a plain form submission. The forms carry
a conventional `method` and `action` alongside the `hx-` attributes, and the
handlers answer accordingly, so the admin remains usable with JavaScript
disabled. HTMX is always detected the same way, by checking whether the
`HX-Request` header equals `true`.

## The public site

The public pages are `index.html` for the homepage, `post.html` for a single post,
`page.html` for a single page, and `not_found.html` behind the themed 404. They
share `layout.html`, which defines the `site_head` and `site_nav` partials, and
they are parsed together as one group so those includes resolve.

What they render comes from `SiteData` in `internal/handler/web/site.go`: the site
title and tagline read from the settings table, whether the visitor is signed in,
and the rows the page is about. The post and page bodies reach the template as
HTML rather than as a string, because the editor stores markup and escaping it
would print the markup instead of rendering the article. That is a deliberate
exception to escaping, and it is only safe because the content comes from the
authenticated admin. Anything else that ever feeds a public page, such as
comments, would have to be sanitised before it got here.

The body carries `hx-boost="true"`, which turns navigation within the theme into a
body swap instead of a full page load. The links that leave the theme opt out with
`hx-boost="false"`: the admin and login pages belong to the other theme, whose
stylesheet a swap would not carry over, and logout answers with JSON rather than
HTML. The themed 404 exists partly because of boosting, since a swap needs an HTML
body and a stale link would otherwise appear to do nothing at all.

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

`GET /admin/settings/` renders `settings.html` plus the `site_settings` include,
both filled with the stored values. The include falls back to the default site
title and tagline when the values map is missing or a key is absent, which is
what a database with no settings rows produces.

The site settings form is a single section with the id `site-settings`, and it
posts to `/admin/settings/site/` both through `action` and through `hx-post`,
targeting itself with `hx-swap="outerHTML"`. Over HTMX the handler answers with
the same section re-rendered from the database, so the saved values appear in
place and the swapped-in markup keeps working for a second save. Without HTMX the
handler redirects to `/admin/settings/` with a `303`, which is the same page
under its canonical URL. Either way the form writes only the whitelisted keys
`site_title` and `tagline`, and empty values are skipped rather than saved.

## Adding a page

Three places have to change together. First, add the template file, and make sure
any `{{template}}` include it uses is parsed into the same group. Second, add the
file to the matching `template.ParseFiles` call in `NewRenderer` and add a
`Renderer` method that executes the right template name. Third, register the
route in `internal/core/router.go` with a trailing slash. If the page needs the
admin shell, the existing pages are the pattern to copy: the same header, nav and
stylesheet link appear at the top of `posts.html`, `pages.html`, `settings.html`,
`dashboard.html` and both editors. A public page is simpler, because
`layout.html` supplies the head and the navigation and the new file only has to
define its body.
