# goFolio CMS – Authentication

Access control is small enough to describe exhaustively: a `users` table, bcrypt
hashes, a signed cookie holding the username and an administrator flag, and one
middleware that guards the `/admin/` route group.

## Users and passwords

Passwords are hashed with bcrypt through `pkg/util`. The work factor is the
library default, exposed as `util.BcryptCost` so it can be raised deliberately in
one place. `HashPassword` generates a per-password salt, and `CheckPasswordHash`
compares a candidate against a stored hash.

`service.CreateUser` inserts a row and hashes the password. Usernames and emails
are unique, so a duplicate username fails with a constraint error, which is what
the service tests assert. `service.GetUserByUsername` returns a
`user not found` error when no row matches, and it copies the hash into the
`Password` field of the model, which is tagged `json:"-"` so it can never be
serialised into a response.

`service.CreateDefaultAdmin` counts the rows where `is_admin = 1`. When that
count is zero it creates `admin` with the password `password` and with the
administrator flag set. Every later call is a no-op, so the default account is
never recreated after it has been changed or deleted. The startup log prints the
default credentials as a reminder to change them.

## Sessions

Sessions come from `gorilla/sessions` with a cookie store, wrapped by
`middleware.Sessions`. The cookie is named `gofolio_session`, lives for 24 hours,
is scoped to `/`, and is marked `HttpOnly`. It is **not** marked `Secure`, which
is the one setting to revisit before exposing the application over HTTPS.

The signing key comes from `AUTH_SECRET`. When that variable is unset,
`main` generates a random 32 byte token at startup, which means every restart
signs out every user. In development that is convenient; anywhere else it is a
surprise, so set `AUTH_SECRET` explicitly.

The session holds two values: `user_id`, which despite its name is the username,
and `is_admin`, a boolean. `IsAuthenticated` returns true only when `is_admin` is
true, so a session without the administrator flag does not grant access even if
it exists. `GetUserID` reads the username back out, which is what the
password change handler uses to find the row to update.

Nothing is re-checked against the database per request. A session that was valid
when it was issued stays valid until it expires, so deleting or demoting a user
does not revoke an existing cookie.

## Login flows

Two endpoints authenticate a user, and both call the same
`service.Login(db, sess, w, r, username, password)`, which looks the user up,
compares the bcrypt hash, and on success stores the username and the admin flag
in a new session cookie.

The HTML form at `POST /admin/login/` is what the browser uses. Over HTMX, a
failure answers 401 with the `login_error` fragment so the message can be swapped
into the page, and a success answers 200 with `HX-Redirect: /admin/`. Without
HTMX, a failure is a 401 with the text `Invalid credentials` and a success is a
303 redirect to `/admin/`. Both cases are handled in `Admin.LoginPost`, and the
template works either way.

The JSON endpoint at `POST /api/auth/login/` takes
`{"username":"...","password":"..."}` and answers `200 {"success":"logged in"}`
or `401 {"error":"Invalid credentials"}`. It never redirects, which is why the
integration tests use it.

`GET /admin/login/` serves the login page and sends an already authenticated
visitor straight to the dashboard. `RequireAuth` has always redirected
unauthenticated visitors there, so the route existing is what closes the loop.

## Logout and session checks

`GET /api/auth/logout/` empties the session values, sets a negative cookie age so
the browser drops the cookie, and answers with JSON. The admin templates link to
it directly, which is why it is a `GET`. That convenience has a cost: any page
that can embed a link to that URL, including a third party site, can end a
user's session. `GET /api/auth/check/` simply reports whether the current
request is authenticated.

## Password changes

`POST /admin/settings/password/` requires a new password of at least eight
characters and a confirmation that matches it. It reads the username from the
session, hashes the new password and updates the matching row through
`service.ChangePassword`, which reports `user not found` when no row was
affected. Failures render a small HTML paragraph with the reason and an
appropriate status code, and the setting page swaps that into the page.

Two things to know about this flow. Existing sessions stay valid after a
password change, because only the hash is replaced; anyone holding a stolen
cookie keeps it until it expires. And `ChangePassword` does not require the
current password, so it relies entirely on the session for authorisation.

## Security posture

What is in place: bcrypt hashing, `HttpOnly` cookies, signed session values,
parameterised SQL everywhere, a whitelist of writable setting keys, and
administrator-only access enforced by middleware on every `/admin/` route except
login and static assets.

What is knowingly not in place yet: the `Secure` cookie flag, session revocation
on password change or user deletion, request throttling on the login endpoint,
and any CSRF protection beyond the session cookie. The application is aimed at
small self-hosted deployments, and anyone exposing it publicly should set
`AUTH_SECRET`, change the default password immediately, terminate TLS in front of
it, and then turn on the `Secure` flag in `middleware.NewSessions`.
