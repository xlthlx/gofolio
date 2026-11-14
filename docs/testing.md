# goFolio CMS – Testing

Tests are part of the definition of done for every Go change. They already cover
the stores, the user service, the session middleware, every handler, and the full
stack through an integration suite. This document describes how they are
organised, which helpers to reuse, and how to run them.

## Running the tests

`make test` runs `go test -v ./...` across the module. Because SQLite support is
compiled in through `mattn/go-sqlite3`, the tests need `CGO_ENABLED=1` and a C
toolchain, and they run in vendor mode like the rest of the module.

Tests that render templates find them by walking up from the working directory,
so they expect to run from inside the repository. Running `go test` for a single
package from the repository root works the same way.

## Layout

Unit tests sit next to the code they cover. Packages that need access to
unexported identifiers use an internal test package, and the handler package
deliberately uses an external one so the tests exercise the same API that
production code calls.

| Location | Package under test | Style |
|---|---|---|
| `internal/store/*_test.go` | `store` | Internal, one test file per source file |
| `internal/service/user_test.go` | `service` | Internal |
| `internal/handler/middleware/*_test.go` | `middleware` | Internal |
| `internal/handler/web/*_test.go` | `web` | External, package `web_test` |
| `test/integration/api_test.go` | The whole stack | External, package `integration_test` |

## Helpers to reuse

Shared scaffolding lives in a `common_test.go` file per package, and the helper
you need is usually already there.

In `internal/store`, `newTestDB` opens a database in `t.TempDir()`, runs
`store.Migrate`, and closes it during cleanup.

In `internal/service`, `newTestDB` opens its own connection with
`_foreign_keys=on`, limits the pool to one connection, and calls
`InitUsersTable`.

In `internal/handler/middleware`, `testSessions` is a session store built from a
fixed secret, and `newSessionRequest` returns a recorder and a request pair.

In `internal/handler/web`, `common_test.go` provides `testRenderer`, built once
from `web.MustRenderer(web.FindAdminThemeDir())`, `testSess`, `newHandlerTestDB`
for a migrated database that can hold users and content, `newTestRouter` for a
trimmed router that mirrors production, and `doJSON` for issuing a request and
capturing the response. `newTestRouter` is not the production router, so a route
that a test exercises has to be registered there as well. Some test files add
their own narrower helpers: `settings_test.go` has `changePasswordRouter` and
`authedCookie`, and `admin_test.go` has `newTestRouterWithPages`.

In `test/integration`, `newTestServer` boots the real stack with
`httptest.NewServer(core.New(db, tpl, sess))` on a throwaway database with the
default administrator created, and returns a client that carries a cookie jar.
`call` issues a request and `decode` unmarshals a response.

## Conventions

Each source file gets a matching test file, so `admin.go` is covered by
`admin_test.go` and `settings_site.go` by `settings_site_test.go`. Scaffolding
that several files need goes into `common_test.go` rather than being duplicated.

Every test creates its own throwaway database under `t.TempDir()` and registers
`t.Cleanup` for the connection, which keeps tests independent and safe to run in
parallel with `-run` filters. There are no shared fixtures: `test/fixtures` is an
empty placeholder directory, and content is seeded through the store or service
layer inside each test.

Assertions are written by hand with `t.Errorf` and `t.Fatalf`, and messages spell
out the expected value, which is what makes a failure readable without opening
the file.

## Integration tests

The integration suite is the only place that exercises the real router, so it is
where middleware ordering and status codes are verified. Two details make that
possible. The client rejects redirect following by returning
`http.ErrUseLastResponse`, so a 303 from `RequireAuth` or from the login form can
be asserted directly rather than being followed to its destination. And the
cookie jar means a login in one request affects the next, which is how
`TestFullPostLifecycle` walks from a failed login through creating, publishing,
fetching and deleting a post.

## What is not covered

There is no browser-driven test, so the HTMX attributes and the TinyMCE editors
are not executed anywhere. The editors are covered from the server side, by
asserting the rendered form and both response paths of a save; the one part no
test can reach is the JavaScript that syncs TinyMCE into the textarea before the
form is sent. `make test` does not enable the race detector, has no coverage
target, and there is no linting step beyond `go vet`, which is expected to stay
clean.
