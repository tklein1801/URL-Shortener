# Quick Start

This guide covers local development with a host-installed Redis instance, the
local CLI build, and the Docker Compose setup.

## Prerequisites

Install Go 1.21 or newer. The CI and container build currently use Go 1.27.1.
For the local workflow, Redis must be available at `localhost:6379`. Docker
users only need Docker Engine with Docker Compose.

## Run the server locally

Start Redis, then run the server from its module directory:

```bash
cd apps/server
cp .env.example .env
go run .
```

The default server URL is `http://localhost:3000`. On the first start, the
server creates `./data/master-token`, prints the master token once, and shows a
security warning. Save that token securely. Later starts reuse the same file and
do not print the token again.

Check that the process and Redis are available:

```bash
curl http://localhost:3000/health
curl http://localhost:3000/ready
```

The local server reads Redis and HTTP settings from `apps/server/.env`. The
default token file is `apps/server/data/master-token` when started from that
directory.

## Build and run the CLI locally

Build the `surl` executable from the repository root:

```bash
mkdir -p bin
go build -o bin/surl ./apps/cli
```

Configure the server URL and token. `config set-token` reads terminal input
without echoing it; it also accepts a protected file or pipe on standard input.

```bash
./bin/surl config set-server http://localhost:3000
./bin/surl config set-token
./bin/surl config show
```

The CLI stores its configuration at
`$XDG_CONFIG_HOME/url-shortener/config.yaml`, or at
`$HOME/.config/url-shortener/config.yaml` when `XDG_CONFIG_HOME` is unset.

Run the complete workflow:

```bash
./bin/surl status
./bin/surl shorten https://example.com
./bin/surl list
./bin/surl open <id>
./bin/surl delete <id>
```

Use `--json` for machine-readable output and `--config <path>` for an explicit
configuration file. For example:

```bash
./bin/surl shorten https://example.com --json
./bin/surl list --json
```

## Run with Docker Compose

From the repository root, build and start Redis and the server:

```bash
docker compose up --build
```

The server is available at `http://localhost:3000`. Redis is reachable only
inside the Compose network. Redis data is stored in the `redis-data` volume, and
the master token is stored in the `server-token` volume.

Read the initial token from the server logs:

```bash
docker compose logs server
```

Copy the generated token into the local CLI configuration:

```bash
./bin/surl config set-server http://localhost:3000
./bin/surl config set-token
```

When the server container is recreated, the same token is reused as long as the
`server-token` volume is retained. Stop the stack while retaining its data with:

```bash
docker compose down
docker compose up
```

To remove the containers and both named volumes, including the token and Redis
data, run:

```bash
docker compose down -v
```

This causes the next server start to generate a new master token.

## Troubleshooting

- If port `3000` is already in use, change `PORT` for a local server or adjust
  the host side of the Compose port mapping in `compose.yml`.
- If `/ready` returns `503`, Redis is unavailable or still starting. Check
  `docker compose ps` and `docker compose logs redis`.
- If management commands return an authentication error, configure the token
  printed on the first server start. The token is never displayed by
  `config show`; it is masked as `********`.
- If the browser cannot be opened by `surl open`, the command prints the full
  redirect URL so it can be opened manually.
- Use `docker compose logs server` to inspect startup failures. Do not delete
  the `server-token` volume unless you intentionally want to rotate the master
  token.
