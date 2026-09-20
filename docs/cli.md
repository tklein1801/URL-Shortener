# CLI reference

Build `surl` from the repository root:

```bash
mkdir -p bin
go build -o bin/surl ./apps/cli
```

Add `bin` to PATH or invoke `./bin/surl`. All commands support `--config <path>`;
network commands support `--timeout 10s`. `--help` works without configuration.

| Command | Behavior |
| --- | --- |
| `surl config set-server <url>` | Validate and save the server URL |
| `surl config set-token` | Read and save a token without terminal echo |
| `surl config show` | Display the URL and a masked token |
| `surl status` | Check liveness, Redis readiness, and authenticated listing |
| `surl shorten <url>` | Print the new ID and complete redirect URL |
| `surl list` | Print ID/target pairs sorted by ID |
| `surl open <id>` | Launch the default browser and print the redirect URL |
| `surl delete <id>` | Delete the link and print its ID |
| `surl version` | Print the embedded build version (`dev` for local builds) |

`set-token` accepts one line from redirected stdin for automation; it never
accepts the token as a positional argument. Interactive prompts go to stderr.
To provision from an already protected token file:

```bash
surl config set-token < /secure/path/master-token
```

## Output

Successful command results go to stdout. Diagnostics go to stderr. Exit status
is `0` on success (including help), and `1` for invalid arguments, configuration,
network, API, or browser-launch failures. Missing deletions report `link not
found`. Tokens and untrusted remote error details are omitted from diagnostics.

All commands producing results support `--json`; help remains ordinary text.
Examples:

```bash
surl shorten https://example.com --json
# {"id":"a1b2c3d4","url":"https://example.com","short_url":"http://localhost:3000/r/a1b2c3d4"}
surl list --json
# {"data":{"a1b2c3d4":"https://example.com"}}
surl status --json
# {"status":"ok"}
surl delete a1b2c3d4 --json
# {"id":"a1b2c3d4"}
surl config show --json
# {"server_url":"http://localhost:3000","token":"********"}
```

Configuration writes return `{"status":"saved"}` in JSON mode. Version returns
`{"version":"dev"}`. `open --json` returns `{"url":"...","opened":true}`;
a launch failure returns `opened:false`, prints a diagnostic, and exits `1`.
The URL remains available as a manual fallback. `open` uses `xdg-open` on Linux,
`open` on macOS, and `rundll32` on Windows. It constructs a public redirect URL
without a management request; the browser receives a 404 if the ID is missing.

The typed API client refuses management redirects and uses the configured base
URL, including an optional proxy path prefix. Specify the final HTTPS origin
instead of an HTTP address that redirects to it.
