# goFolio CMS – Database

goFolio stores everything in one SQLite file. There is no migration tool, no
schema version table and no second database. This document describes the schema,
how it is created, what the store types do, and what to watch out for when
changing it.

## The file and the connection

`store.Open` takes the path from `DATABASE_PATH`, appends `?_foreign_keys=on`,
and then calls `SetMaxOpenConns(1)`. The single connection is deliberate: SQLite
serialises writes, and a modest pool under concurrent requests is the quickest
way to see `database is locked`. Keep that setting unless the storage layer is
redesigned around it.

`store.Migrate` then runs one `CREATE TABLE IF NOT EXISTS` block. It runs on
every start and is safe to re-run, which is what makes the schema self-healing
for a fresh database file.

## Schema

| Table | Columns |
|---|---|
| `posts` | `id`, `title`, `slug` (unique), `content`, `status` (default `draft`), `created_at`, `updated_at` |
| `pages` | `id`, `title`, `slug` (unique), `content`, `status` (default `draft`), `created_at`, `updated_at` |
| `users` | `id`, `username` (unique), `email` (unique), `password_hash`, `is_admin`, `created_at`, `updated_at` |
| `settings` | `key` (primary key), `value` |

Timestamps default to `CURRENT_TIMESTAMP`. Posts and pages are close enough to
share a shape, and `users` exists purely for authentication. Content has no
author or category relation, and there is no foreign key anywhere in the schema
today.

`status` is free text rather than an enum, and only two values are used in
practice: `draft` and `published`. It defaults to `draft`, and the admin marks a
post or page published through the editor select. An empty status is stored as
`draft` rather than as an empty string, and that rule lives in the store, so
creation and update both apply it and no caller has to remember to. For both
tables it decides public visibility: the homepage lists published posts, and a
post or page that is not published is a 404 at its public URL, so a draft stays
private until it is published. The admin is unaffected, since it lists and loads
rows whatever their status by id. Both `PageStore` and `PostStore` read and write
`status` and `updated_at`, so the JSON for a page reports the stored status and
the time of its last update.

## Migrations

The whole schema lives in one string constant in `internal/store/db.go`. There is
no version table, so an existing database is brought forward by re-running that
block. Adding a new table is easy, because `CREATE TABLE IF NOT EXISTS` is
idempotent. Adding a column is not: SQLite has no `ADD COLUMN IF NOT EXISTS`, so
an `ALTER TABLE` statement placed in the blob fails the second time the server
starts with a duplicate column error. A column addition therefore needs either a
guard that inspects `PRAGMA table_info` first, or a small migration step outside
the idempotent block. Renaming or retyping a column is a manual operation on a
live database.

Schema changes are expected to be tolerant of existing rows, because local
development databases and the one in the Docker volume both survive deployments.

## Store types

Stores are plain structs holding `*sql.DB`, created inline by handlers. There are
no interfaces and no repositories, so a store method is the whole data access
story for its table.

`PostStore` offers `ListPublished`, `ListAll`, `GetPublishedBySlug`, `GetByID`,
`Create`, `Update` and `Delete`. The published variants filter on
`status = 'published'`, and both list methods order by `created_at DESC`.
`GetPublishedBySlug` is the one a public request uses, which is why an unfetched
draft looks like a missing page to the outside world.

`PageStore` offers `ListAll`, `GetPublishedBySlug`, `GetByID`, `Create`, `Update`
and `Delete`. `GetPublishedBySlug` filters on `status = 'published'` exactly as
the post version does, so the two resources share one visibility rule, and the
admin reaches a draft through `GetByID` instead.

`SettingStore` offers `All`, `Get` and `Set`. `All` returns a map keyed by
setting name for template convenience, and `Set` is an upsert written as
`INSERT ... ON CONFLICT(key) DO UPDATE`. `Get` converts `sql.ErrNoRows` into its
own `ErrSettingNotFound` sentinel, which is the one place the package does not
pass the standard error through.

## Conventions and side effects

`sql.ErrNoRows` is returned unchanged by the content stores, and handlers map it
to a 404 with `errors.Is`. That keeps "no such row" separate from "the query
failed".

`Create` writes the row, reads back the inserted id, and fills `ID`, `CreatedAt`
and `UpdatedAt` on the struct it was given. `Update` takes the id as a separate
argument rather than reading it from the struct, and for posts it also sets
`updated_at` to the current time. `Delete` does not check how many rows were
affected, so deleting an id that does not exist succeeds silently.

Slugs are declared unique, and nothing in the write path deduplicates them. If
two posts end up with the same title and no explicit slug, the second insert
fails on the constraint, and the API answers with a 500 carrying the raw SQLite
message. Callers that need a different error would have to inspect the
constraint.

All queries use parameter placeholders, and the settings handler additionally
limits writes to a whitelist of keys, so no user input reaches SQL as text.

## Inspecting and resetting

The database is an ordinary SQLite file, so `sqlite3 gofolio.db` with `.tables`
and `.schema` is enough to inspect it. `make seed` creates the file and the
default administrator without starting a server, and `make clean` deletes every
`*.db` file in the repository root, which wipes all content. To back the
database up safely, stop the server and copy the file, or use the `sqlite3`
`.backup` command while it is running.
