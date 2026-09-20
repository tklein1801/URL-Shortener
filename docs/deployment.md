# Deployment

## Docker

Build from the server directory (or use it as the build context):

```bash
docker build -t url-shortener-server apps/server
```

The multi-stage Dockerfile pins Go to a patch release, compiles a static binary,
and uses a minimal `scratch` runtime with CA certificates. The service runs as
UID/GID `10001:10001`; `/data` belongs to that user and has mode `0700`.

Run one server instance with a durable volume:

```bash
docker volume create surl-data
docker run -d --name surl-server \
  -p 127.0.0.1:3000:3000 \
  -e SQLITE_PATH=/data/links.db \
  -e MASTER_TOKEN_FILE=/data/master-token \
  -v surl-data:/data \
  url-shortener-server
```

The SQLite database and master token share `/data`. SQLite uses WAL journaling,
so the directory must remain writable for the database and its sidecar files.
Run exactly one server container for each database file; multiple replicas and
network filesystems are not supported.

Read the initial token from `docker logs surl-server` and store it securely.
Container logs contain this first-start secret: restrict log access and retention.
Configure the CLI with `surl config set-token`. Subsequent restarts do not print
the token. Keep mounting the same named volume when recreating the container.
Never bake a token, database, or `.env` file into the image.

For bind mounts, create a dedicated directory owned by UID 10001 with mode
`0700`. The server must be able to create, chmod, lock, and sync files there.

## Compose upgrades and backups

The included Compose file runs only the server. Its existing `server-token`
volume name is retained for upgrade compatibility, but the volume now contains
both `master-token` and `links.db`. The former Redis volume is no longer mounted;
existing Redis links are not imported or deleted.

For a raw volume backup, stop the server first so SQLite closes and checkpoints
its WAL, then back up the complete data volume. Restore the token and database
together. Removing the volume rotates the management token and deletes all
SQLite links.

## Networking and HTTPS

Use HTTPS through a reverse proxy for remote clients. The application itself
serves plain HTTP and should be reachable only through trusted local/container
networks in such a deployment. Preserve the Authorization header and route
`/api/v1/urls`, `/r/`, `/health`, and `/ready`. For a public prefix such as `/surl`,
strip that prefix upstream and configure the CLI with the prefixed external URL.

Use `/health` for process liveness and `/ready` for SQLite readiness. A live
server returns `503` for readiness and link operations if the database becomes
unavailable. The scratch image has no shell or curl; perform HTTP probes from the
orchestrator or an external probe client. Allow at least the configured shutdown
timeout when stopping a container.

The master token grants full create/list/delete access. There are no per-user
roles, expiry policies, or rate limits. Redirects intentionally remain public.
