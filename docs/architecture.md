# Architecture

The Go workspace contains the independent modules `apps/server` and `apps/cli`.
Each application has a small `main.go` composition root and internal packages.
The API contract is documented in [api.md](api.md); each module owns its wire
types so both applications can be built independently.

## Server

```text
HTTP request -> httpapi -> links.Service -> links.Repository
                                                ^
                                                |
                                         sqlitestore.Store -> SQLite
```

- `config` loads `.env` and environment settings and validates runtime options.
- `auth` creates or loads the master token and verifies SHA-256 digests using a
  constant-time comparison. Raw token lengths do not affect digest comparison.
- `links` owns URL validation, random ID generation, collision retries, the
  repository interface, and application errors.
- `sqlitestore` owns the SQLite connection and implements the repository interface.
- `httpapi` owns routing, authentication middleware, JSON responses, and the
  mapping from application errors to HTTP statuses.

`Repository` exposes `List`, `Get`, `Create`, and `Delete`, each accepting a
request context. Its errors include `ErrNotFound`, `ErrConflict`, and
`ErrBackend`; cancellation remains distinguishable. `Delete` returns whether an
entry existed. The HTTP layer never imports the SQLite driver.

A replacement persistence adapter implements this interface, plus lifecycle and
readiness operations supplied by the composition root. The service and handlers
remain unchanged. Tests use both a fake repository and temporary SQLite files.

## SQLite persistence

The `links` table stores the short ID as its primary key and the original URL as
a required text value. The schema is created automatically at startup. Creation
uses an insert that cannot overwrite an existing ID; collisions cause the service
to generate a new ID, up to 16 attempts. IDs encode six random bytes as eight
URL-safe base64 characters.

The adapter uses one database connection and enables WAL journaling, full
synchronous durability, and a bounded busy timeout. The deployment model is one
server instance per local database file. Backend failures abort operations rather
than masquerading as missing links. Existing Redis data is not migrated.

## Lifecycle

Startup validates configuration, resolves the token, opens and initializes SQLite, then starts
an `http.Server`. HTTP and backend calls have bounded timeouts. SIGINT or SIGTERM
initiates graceful HTTP shutdown; if the deadline expires, connections close.
The SQLite connection closes after HTTP shutdown. Authentication state, clients, and
contexts are passed explicitly, with no package-global request state.

## CLI

`command` constructs Cobra commands with injectable input, output, token input,
browser launch, and API factory dependencies. `config` owns XDG paths and atomic
YAML persistence. `apiclient` parses the base URL once, adds bearer authentication,
handles typed responses and errors, and uses an injected HTTP client.

Management redirects are not followed, preventing credentials from being sent
to a redirected endpoint. Diagnostics omit transport error details and remote
error messages that could echo secrets. Browser errors print the redirect URL
and return a failure exit code.
