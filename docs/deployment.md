# Deployment

## Docker

Build from the server directory (or use it as the build context):

```bash
docker build -t url-shortener-server apps/server
```

The multi-stage Dockerfile pins Go to a patch release, compiles a static binary,
and uses a minimal `scratch` runtime with CA certificates. The service runs as
UID/GID `10001:10001`; `/data` belongs to that user and has mode `0700`.

Create durable volumes and a private network, then start Redis and the server:

```bash
docker network create surl
docker volume create surl-redis
docker volume create surl-token
docker run -d --name surl-redis --network surl \
  -v surl-redis:/data redis:7.4.2-alpine redis-server --appendonly yes
docker run -d --name surl-server --network surl \
  -p 127.0.0.1:3000:3000 \
  -e REDIS_HOST=surl-redis:6379 \
  -e MASTER_TOKEN_FILE=/data/master-token \
  -v surl-token:/data \
  url-shortener-server
```

Redis is not published to the host. Adjust the Redis version and authentication
to your deployment requirements. If Redis requires a password, provide matching
`REDIS_PW` to the server. A dedicated Redis database is required because all
string keys are treated as links. The startup Redis check must succeed before
HTTP requests can be served.

Read the initial token from `docker logs surl-server` and store it securely.
Container logs contain this first-start secret: restrict log access and retention.
Configure the CLI with `surl config set-token`. Subsequent restarts do not print
the token. Keep mounting the same **named volume** when recreating containers;
an anonymous volume is insufficient for reliable token reuse across recreation.
Never bake a token or `.env` file into an image.

For bind mounts, create a dedicated directory owned by UID 10001 with mode
`0700`. The server must be able to write, chmod, hard-link, and sync files there.
Do not mount a read-only token file: startup also enforces its permissions.
Back up both Redis data and the token volume. Removing the token volume changes
the management credential on the next start; it does not remove Redis links.

## Networking and HTTPS

Use HTTPS through a reverse proxy for remote clients. The application itself
serves plain HTTP and should be reachable only through trusted local/container
networks in such a deployment. Preserve the Authorization header and route
`/api/v1/urls`, `/r/`, `/health`, and `/ready`. For a public prefix such as `/surl`,
strip that prefix upstream and configure the CLI with the prefixed external URL.

Use `/health` for process liveness and `/ready` for Redis readiness. A live
server returns `503` for readiness and link operations during backend outages.
The scratch image has no shell or curl; perform HTTP probes from the orchestrator
or an external probe client. Allow at least the configured shutdown timeout when
stopping a container (Docker's `--stop-timeout` can be increased).

The master token grants full create/list/delete access. There are no per-user
roles, expiry policies, or rate limits. Redirects intentionally remain public.
