# Go URL Shortener

A Redis-backed HTTP service and the `surl` command-line client, maintained as two
modules in a Go workspace. Management operations require a bearer master token;
redirects and health checks are public.

## Quick start

See the [English quick-start guide](docs/quick-start.md) for local server and
CLI commands, Docker Compose, token setup, and troubleshooting.

## Documentation

- [Architecture](docs/architecture.md): application layers and persistence adapters
- [Quick start](docs/quick-start.md): local development and Docker Compose
- [API](docs/api.md): routes, authentication, schemas, and errors
- [CLI](docs/cli.md): commands and output examples
- [Configuration](docs/configuration.md): environment, files, and permissions
- [Deployment](docs/deployment.md): Docker, Redis, token persistence, and HTTPS
- [Development](docs/development.md): tests, workspace, CI, and releases

The versioned API replaces the former `/list`, `/shorten`, and `/d/{id}` routes.
Query-parameter authentication and the old CLI configuration are no longer used.
Existing Redis link entries remain readable without a migration.

## Validation

```bash
go test -race ./apps/server/... ./apps/cli/...
go vet ./apps/server/... ./apps/cli/...
```

Tests include an in-process Redis implementation, HTTP contract checks, secure
file handling, isolated CLI commands, and a complete CLI-to-server workflow.
