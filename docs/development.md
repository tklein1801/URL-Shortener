# Development

## Workspace

`go.work` contains `apps/server` (`url-shortener`) and `apps/cli`
(`url-shortener-cli`). Modules retain Go 1.21 language compatibility; CI and
container builds use Go 1.27.1. The modules have independent `go.mod`/`go.sum`
files, and each can be built from its own directory with `GOWORK=off`.

From the repository root:

```bash
go test -race ./apps/server/... ./apps/cli/...
go vet ./apps/server/... ./apps/cli/...
gofmt -l apps/server apps/cli
mkdir -p bin
go build -o bin/server ./apps/server
go build -o bin/surl ./apps/cli
```

Use explicit module patterns; `go test ./...` at the workspace root does not
identify either module. Run `go mod tidy` separately in each module after changing
dependencies. No configuration or network access occurs during package init.

## Test coverage

- Application tests use a fake repository to check validation, collision retries,
  missing IDs, and cancellation.
- Adapter tests use temporary SQLite files and cover schema creation, conflicts,
  listing, deletion, persistence across reopen, cancellation, and backend failures.
- Auth tests cover 256-bit token generation, reuse, permissions, malformed files,
  unreadable files, symlinks, failures, and concurrent initial starts.
- HTTP tests use `httptest` to exercise public and authenticated routes, JSON
  contracts, invalid input, missing IDs, and backend outage behavior.
- CLI configuration tests isolate XDG/home paths with temporary directories and
  verify strict validation, permissions, and atomic inode replacement.
- API client tests use local HTTP servers for authentication, path construction,
  response decoding, redaction, cancellation, redirects, and timeouts.
- Command tests inject an API and browser launcher, checking deterministic/JSON
  output, masking, help without side effects, and exit codes.
- The server's end-to-end test builds the real CLI and runs configuration,
  status, shortening, listing, redirecting, opening, and deleting against the
  full server HTTP stack and SQLite. Linux browser opening uses a local stub.
- A server process test checks first-start token output, silent token reuse,
  SQLite link persistence across restart, readiness, and graceful shutdown.

Tests need permission to open loopback sockets. `go test -short` skips the CLI
binary build/end-to-end test. The unreadable-file test requires a non-root Unix
user. Platform-specific browser and terminal integration should also be checked
on the target operating system when changing those integrations.

## CI and releases

Concourse pipelines under [`.ci/`](../.ci/) run formatting checks, `go vet`,
race-enabled tests, and both application builds for `main`, `feat/*`,
`feature/*`, `fix/*`, and `refactor/*` branches. The main pipeline then
cross-compiles the CLI for Linux, macOS, and Windows on amd64 and arm64,
publishes a deterministic GitHub release, and pushes the server image to
GHCR for Linux amd64 and arm64 for every successful commit.

Pull requests targeting `main` are tested as a merge with the current target
branch. The PR job runs the same checks and builds and reports the result through
the `concourse/pr-main` GitHub status context.

Every successful `main` commit publishes a deterministic release named and
tagged `surl-YYYYMMDDTHHMMSSZ-<12-character-commit-sha>`. The version is
embedded in each CLI binary. The server image receives both `latest` and the
deterministic release tag. GitHub credentials are provided through the
Concourse credential manager. See [`.ci/README.md`](../.ci/README.md) for
pipeline installation, worker requirements, Vault variables, and release
operations.
