# Go URL Shortener

A small URL shortener written in Go. The project contains a Redis-backed HTTP
server and a command-line client for creating and managing shortened URLs.

## Current status

The project currently provides:

- A Go 1.21 HTTP server in `apps/server`
- Redis storage for short URL mappings
- A `surl` command-line client in `apps/cli`
- A Dockerfile for the server
- A GitHub Actions workflow that builds and releases CLI binaries for Linux,
  macOS, and Windows

There are currently no automated tests. The application is a small, single-
process service and does not currently include user accounts, per-link
expiration, TLS termination, or a database migration system.

## Repository structure

```text
.
├── apps/
│   ├── cli/       # Command-line client
│   └── server/    # HTTP server and Dockerfile
├── .github/
│   └── workflows/ # Release automation
└── go.work        # Go workspace containing both applications
```

## Requirements

- Go 1.21 or newer
- A running Redis instance
- Docker, if the server is run as a container

The server requires Redis to be reachable before it starts handling requests.
The Redis password must be configured consistently in both Redis and the
server environment. The current implementation requires `REDIS_PW` to be
non-empty, even when Redis is running locally.

## Run the server locally

The server loads `.env` from its current working directory. Run these commands
from `apps/server`:

```bash
cd apps/server
cp .env.example .env
```

Edit `.env` with values matching your Redis instance:

```dotenv
REDIS_HOST=localhost:6379
REDIS_PW=redis-password
REDIS_DB=0
CODE=change-this-auth-code
```

The variables are:

| Variable | Required | Description |
| --- | --- | --- |
| `REDIS_HOST` | Yes | Redis host and port, for example `localhost:6379` |
| `REDIS_PW` | Yes | Redis password |
| `REDIS_DB` | Yes | Redis database number, for example `0` |
| `CODE` | Yes | Authentication code used by protected endpoints |
| `PORT` | No | HTTP port; defaults to `3000` |

Start the server with:

```bash
go run .
```

The server listens on `http://localhost:3000` by default. The health endpoint
is available at `http://localhost:3000/health`.

## Run with Docker

Build the image from the server directory:

```bash
cd apps/server
docker build -t url-shortener-server .
```

Run the container with the environment file and expose port 3000:

```bash
docker run --rm \
  --env-file .env \
  -p 3000:3000 \
  url-shortener-server
```

When Redis runs in another container, both containers must be on the same
Docker network and `REDIS_HOST` must use the Redis container or service name
instead of `localhost`.

## API

The server exposes the following endpoints. The authentication code is passed
as the `code` query parameter for protected endpoints.

### Health check

```http
GET /health
```

Returns HTTP `200` when the server is running.

### List shortened URLs

```http
GET /list?code=<AUTH_CODE>
```

Requires the authentication code. The response contains all key/value pairs
stored in Redis:

```json
{
  "data": {
    "a1b2c3d4": "https://example.com"
  }
}
```

### Create a shortened URL

```http
POST /shorten
Content-Type: application/x-www-form-urlencoded

url=https%3A%2F%2Fexample.com
```

The endpoint does not currently require authentication. A successful request
returns a randomly generated eight-character identifier:

```json
{
  "shortUrl": "a1b2c3d4"
}
```

Example:

```bash
curl -X POST http://localhost:3000/shorten \
  --data-urlencode 'url=https://example.com'
```

Returns `400 Bad Request` when the `url` form field is missing.

### Redirect to the original URL

```http
GET /r/<SHORT_URL>
```

Looks up the identifier and redirects to the stored URL with HTTP `307
Temporary Redirect`. This endpoint does not currently require authentication.
It returns `404 Not Found` when the identifier cannot be found.

Example:

```bash
curl -i http://localhost:3000/r/a1b2c3d4
```

### Delete a shortened URL

```http
DELETE /d/<SHORT_URL>?code=<AUTH_CODE>
```

Requires the authentication code. A successful request returns the deleted
identifier:

```json
{
  "shortUrl": "a1b2c3d4"
}
```

Example:

```bash
curl -X DELETE \
  'http://localhost:3000/d/a1b2c3d4?code=<AUTH_CODE>'
```

## Command-line client

The CLI executable is named `surl`. Run it from `apps/cli` during development:

```bash
cd apps/cli
go run . --help
```

To build a local executable:

```bash
go build -o surl .
```

The CLI stores its YAML configuration at:

```text
~/.config/surl/config.yml
```

Configure the server and authentication code:

```bash
./surl set http://localhost:3000 <AUTH_CODE>
```

The equivalent individual commands are:

```bash
./surl set-host http://localhost:3000
./surl set-code <AUTH_CODE>
```

Available commands:

| Command | Alias | Description |
| --- | --- | --- |
| `set <host> <code>` | | Set the host URL and authentication code |
| `set-host <host>` | | Set the host URL |
| `get-host` | | Display the configured host URL |
| `set-code <code>` | | Set the authentication code |
| `get-code` | | Display the configured authentication code |
| `list` | `ls` | List all shortened URLs |
| `shorten <url>` | `create` | Create a shortened URL |
| `open <id>` | `o` | Open a shortened URL in the default browser |
| `delete <id>` | `del` | Delete a shortened URL |

Examples:

```bash
./surl shorten https://example.com
./surl list
./surl open a1b2c3d4
./surl delete a1b2c3d4
```

## Development

The repository uses a Go workspace containing both modules. Dependencies can be
downloaded and each application can be checked independently:

```bash
cd apps/server
go mod download
go test ./...

cd ../cli
go mod download
go test ./...
```

There are currently no `_test.go` files, so these commands primarily verify
that the respective modules compile.

## Releases

The workflow in `.github/workflows/release.yml` can be started manually or is
triggered by pushes to `main` that change `apps/cli/main.go`. It builds CLI
artifacts for supported Linux, macOS, and Windows architectures, creates a
timestamped Git tag, and publishes a GitHub release containing the binaries.

The workflow only covers CLI releases. The server is not automatically
published by the release workflow.
