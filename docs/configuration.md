# Configuration

## Server

The server loads `.env` from its working directory. Existing environment variables
take precedence over `.env`; empty optional settings use their defaults. Invalid
ports, database numbers, addresses, and nonpositive durations fail startup.

| Variable | Default | Purpose |
| --- | --- | --- |
| `REDIS_HOST` | `localhost:6379` | Redis address as host:port |
| `REDIS_PW` | empty | Optional Redis password |
| `REDIS_DB` | `0` | Nonnegative Redis database index |
| `PORT` | `3000` | HTTP port, 1–65535 |
| `MASTER_TOKEN_FILE` | `./data/master-token` | Durable master token path |
| `HTTP_READ_TIMEOUT` | `10s` | Header and request read timeout |
| `HTTP_WRITE_TIMEOUT` | `15s` | Response write timeout |
| `HTTP_IDLE_TIMEOUT` | `60s` | Keep-alive idle timeout |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful HTTP shutdown deadline |
| `BACKEND_TIMEOUT` | `5s` | Startup Redis check and request context deadline |

Duration settings use Go syntax such as `500ms`, `5s`, or `1m`. Relative token
paths resolve from the server working directory. `CODE` is no longer used.

## Master token lifecycle

A missing file causes generation of 32 cryptographically random bytes, encoded
as 43 unpadded base64url characters (256 bits of entropy). The server writes and
syncs a temporary file, then publishes it using an exclusive hard link. This
requires a filesystem supporting hard links and directory syncing. Concurrent
starters reuse the winner's complete token. An existing file is never replaced.

The parent directory is secured to `0700`, and the token file to `0600`. Use a
dedicated private directory, including with custom paths: the immediate parent
is chmod-ed even when it already exists. Files must be regular, readable, and
contain the canonical 43-character token with at most one final newline.
Symlinks, empty files, malformed files, and unreadable files fail startup.

Only the creator prints the token to stdout. Restarts read and reuse it silently.
The startup warning explains its management authority and one-time display.
Capture the initial output securely; if necessary, an authorized administrator
can read the protected file later. A failure after file creation may leave the
valid token file in place, which is deliberately reused on the next start.
Back up the token alongside deployment data. Replacing or removing it is an
explicit administrative token rotation and invalidates configured clients.

## CLI

Path precedence:

1. `--config <path>` (relative to the CLI working directory if relative).
2. `$XDG_CONFIG_HOME/url-shortener/config.yaml` if XDG_CONFIG_HOME is nonempty.
3. `$HOME/.config/url-shortener/config.yaml` otherwise.

XDG_CONFIG_HOME must be absolute. A missing file uses the server default below
and an empty token in memory. Help, version, show, and ordinary commands never
create the configuration; only successful `config set-*` commands write it.

```yaml
server_url: http://localhost:3000
token: your-generated-master-token
```

YAML is validated strictly; unknown fields and duplicate keys fail. Server URLs
must be absolute HTTP(S) URLs with a hostname and without embedded credentials,
queries, or fragments. A reverse-proxy path prefix is supported. Tokens must be
nonempty printable ASCII without whitespace when set; an empty token is allowed
in a partially configured file but cannot authenticate management requests.

Writes create a `0600` temporary file in the destination directory, sync it, then
atomically rename it into place. The immediate configuration directory is secured
to `0700`, including with custom paths; choose a dedicated private directory.
`config show` always masks a configured token as `********`.

POSIX permission modes apply on Unix. On Windows, protect the configuration
folder with the user's ACL; chmod does not provide equivalent POSIX isolation.
The CLI never migrates or reads the former `~/.config/surl/config.yml` file.

`--timeout` controls the HTTP client and command deadline (default `10s`, strictly
positive). There are no implicit CLI environment overrides for URL or token.
