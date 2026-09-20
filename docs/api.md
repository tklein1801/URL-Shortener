# HTTP API

Base URL in these examples: `http://localhost:3000`. Management endpoints require
`Authorization: Bearer <TOKEN>`. The authentication scheme is case-insensitive;
the token is case-sensitive. Invalid or absent credentials return `401` with
`WWW-Authenticate: Bearer`. A `code` query parameter never authenticates a request.

| Method | Path | Auth | Success |
| --- | --- | --- | --- |
| GET | `/health` | Public | `200 {"status":"ok"}` |
| GET | `/ready` | Public | `200 {"status":"ok"}` if SQLite responds |
| GET | `/api/v1/urls` | Bearer | `200` with all links |
| POST | `/api/v1/urls` | Bearer | `201` with the new link |
| DELETE | `/api/v1/urls/{id}` | Bearer | `200` with the deleted ID |
| GET | `/r/{id}` | Public | `307` with the original URL in `Location` |

## Create

```http
POST /api/v1/urls
Authorization: Bearer <TOKEN>
Content-Type: application/json

{"url":"https://example.com/path?query=value"}
```

```json
{"id":"a1b2c3d4","url":"https://example.com/path?query=value"}
```

The response includes `Location: /r/a1b2c3d4`. Construct the full redirect URL
from the configured external base URL; the server does not trust incoming Host
headers to construct absolute URLs. Only absolute `http` or `https` targets with
a hostname are accepted. JSON must contain one object with a `url` field and no
unknown fields. The request body is limited to 1 MiB. IDs have eight URL-safe
characters. Existing IDs cannot be overwritten by creation.

## List and delete

A list response contains the complete ID-to-URL map, or an empty map:

```json
{"data":{"a1b2c3d4":"https://example.com/path?query=value"}}
```

A successful deletion returns:

```json
{"id":"a1b2c3d4"}
```

Deleting an already missing ID returns `404`. Redirects require no token and
return `404` only if the ID is missing.

## Errors

Error responses have `Content-Type: application/json` and this shape:

```json
{"error":{"code":"not_found","message":"link not found"}}
```

| Status | Code | Meaning |
| --- | --- | --- |
| 400 | `invalid_request` | Malformed, oversized, or unexpected JSON |
| 400 | `invalid_url` | Target is not an absolute HTTP(S) URL |
| 401 | `unauthorized` | Missing or invalid bearer token |
| 404 | `not_found` | Link or route does not exist |
| 405 | `method_not_allowed` | Unsupported method for this route |
| 503 | `unavailable` | Backend failure, timeout, or exhausted collision retries |
| 500 | `internal_error` | Unexpected handler panic |

`/health` reports process liveness independently of SQLite. `/ready` returns
`503` if the database is unavailable. Storage details are never included in
HTTP errors. Removed legacy routes return `404`.
