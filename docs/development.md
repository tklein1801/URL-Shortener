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
- Adapter tests use miniredis over local TCP, including a deliberately verified
  multi-page SCAN, legacy entries, SET NX, deletion, and backend failures.
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
  full server HTTP stack and miniredis. Linux browser opening uses a local stub.
- A server process test checks first-start token output, silent token reuse on
  restart, readiness, and successful shutdown after a termination signal.

Tests need permission to open loopback sockets. `go test -short` skips the CLI
binary build/end-to-end test. The unreadable-file test requires a non-root Unix
user. Platform-specific browser and terminal integration should also be checked
on the target operating system when changing those integrations.

## CI and releases

CI runs formatting checks, `go vet`, race-enabled tests for both modules, and a
Docker build. The release workflow runs the same checks before cross-compiling
Linux, macOS, and Windows binaries for amd64/arm64, plus Linux arm.

Push an explicitly chosen `v*` tag to publish a release. The tag is embedded
using `-ldflags "-X main.version=<tag>"`. A manual workflow dispatch builds
artifacts with the commit SHA as version but does not publish a release or create
a tag. Publishing uses the workflow's scoped `GITHUB_TOKEN`; no personal access
token is needed. Releases include separately named executable artifacts.

Version references: [Go downloads](https://go.dev/dl/),
[checkout releases](https://github.com/actions/checkout/releases),
[setup-go releases](https://github.com/actions/setup-go/releases),
[upload-artifact releases](https://github.com/actions/upload-artifact/releases),
[download-artifact releases](https://github.com/actions/download-artifact/releases).
