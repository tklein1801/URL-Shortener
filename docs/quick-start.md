# Quick Start

This guide covers local development, the command-line client, and Docker Compose.

## Prerequisites

Install Go 1.21 or newer. The CI and container build currently use Go 1.27.1.
SQLite is embedded in the server, so no separate database service is required.
Docker users only need Docker Engine with Docker Compose.

## Run the server locally

From the server module directory:

```bash
cd apps/server
cp .env.example .env
go run .
```

The default server URL is `http://localhost:3000`. On the first start, the
server creates `./data/master-token` and `./data/links.db`, prints the master
token once, and shows a security warning. Save that token securely. Later starts
reuse the same files and do not print the token again.

Check process and database readiness:

```bash
curl http://localhost:3000/health
curl http://localhost:3000/ready
```

The local server reads SQLite and HTTP settings from `apps/server/.env`.
Relative paths are resolved from the server working directory.

## Build and run the CLI locally

Build the `surl` executable from the repository root:

```bash
mkdir -p bin
go build -o bin/surl ./apps/cli
```

Configure the server URL and token. `config set-token` reads terminal input
without echoing it and also accepts a protected file or pipe on standard input.

```bash
./bin/surl config set-server http://localhost:3000
./bin/surl config set-token
./bin/surl config show
```

The CLI stores its configuration at
`$XDG_CONFIG_HOME/url-shortener/config.yaml`, or at
`$HOME/.config/url-shortener/config.yaml` when XDG_CONFIG_HOME is unset.

Run the complete workflow:

```bash
./bin/surl status
./bin/surl shorten https://example.com
./bin/surl list
./bin/surl open <id>
./bin/surl delete <id>
```

Use `--json` for machine-readable output and `--config <path>` for an explicit
configuration file.

## Run with Docker Compose

From the repository root:

```bash
docker compose up --build
```

The server is available at `http://localhost:3000`. The SQLite database and
master token are stored in the `server-token` volume. The historical volume name
is retained so existing Compose installations keep their master token.

Read the initial token from the server logs:

```bash
docker compose logs server
```

Stop and restart while retaining data with:

```bash
docker compose down
docker compose up
```

To remove the container, token, and all shortened links, run:

```bash
docker compose down -v
```

For backups, stop the server and copy the complete volume. Existing Redis data
is not migrated or used by this version.

## Troubleshooting

- If port `3000` is already in use, change `PORT` locally or adjust the host side
  of the Compose port mapping.
- If `/ready` returns `503`, check that `SQLITE_PATH` points to a writable local
  directory and inspect `docker compose logs server`.
- If management commands return an authentication error, configure the token
  printed on the first server start. `config show` always masks it.
- If the browser cannot be opened by `surl open`, use the redirect URL printed by
  the command.
- Do not delete the data volume unless you intentionally want to remove links and
  rotate the master token.
